//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2017 Rui NI <ranqus@gmail.com>
//
//  This file is part of Crypto-Obscured Forwarder.
//
//  Crypto-Obscured Forwarder is free software: you can redistribute it
//  and/or modify it under the terms of the GNU General Public License
//  as published by the Free Software Foundation, either version 3 of
//  the License, or (at your option) any later version.
//
//  Crypto-Obscured Forwarder is distributed in the hope that it will be
//  useful, but WITHOUT ANY WARRANTY; without even the implied warranty
//  of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with Crypto-Obscured Forwarder. If not, see
//  <http://www.gnu.org/licenses/>.

package join

import (
	"errors"
	"io"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/projector/projection"
)

// Errors
var (
	ErrProcessorReceivingMulitpleClientsReceived = errors.New(
		"Receiving mulitple clients. Which usually means the previous " +
			"client hasn't been handled properly")

	ErrProcessorReceivingStopAlreadyClosed = errors.New(
		"Receiving already been closed, it can't be stop again")

	ErrProcessorWaitUnexpectedClientRelayInitedSignal = errors.New(
		"Unexpected RespondClientRelayInitialized Signal")

	ErrProcessorWaitUnexpectedClientRelayInitFailureSignal = errors.New(
		"Unexpected RespondClientRelayInitializationFailed Signal")

	ErrProcessorWaitNotReadyToHandleClientPingRequest = errors.New(
		"Not ready for Ping request from Client. This usually means " +
			"we having a depending PingRequest waiting for respond")

	ErrProcessorWaitUnexpectedClientPingRespond = errors.New(
		"Client has sent an unexpected Ping Respond")

	ErrProcessorWaitUnknownRespondReceived = errors.New(
		"Unknown respond has been received during waiting")

	ErrProcessorAccessorClosedForcely = errors.New(
		"Accessor has been forcely closed")

	ErrProcessorAccessorRelayFailed = errors.New(
		"Accessor has been closed because Relay has failed to " +
			"initialize the handling procedure")

	ErrProcessorWaitRelayInitFailed = errors.New(
		"Remote Relay has failed to initialized")
)

// Responds that client send to us
const (
	RespondJoined                          = 0x00
	RespondJoinErrorNotFound               = 0x01
	RespondJoinErrorInternalFailure        = 0x02
	RespondClientRelayInitialized          = 0x10
	RespondClientRelayInitializationFailed = 0x11
	RespondClientPingRequest               = 0x20
	RespondClientPingRespond               = 0x22
	RespondClientRelease                   = 0x23
	RespondClientQuit                      = 0x25
)

// Requests that will be send to Client
const (
	RequestClientRelayRequest  = 0x00
	RequestClientPingEmit      = 0x21
	RequestClientRelease       = 0x24
	RequestClientReleaseCancel = 0x26
)

// processor Join request processor
type processor struct {
	logger                      logger.Logger
	cfg                         Config
	runner                      worker.Runner
	registered                  registerations
	rw                          rw.ReadWriteDepleteDoner
	currentProjectionID         projection.ID
	currentReceiveResult        chan error
	currentReceiver             projection.Receiver
	currentReceivedAccessorChan chan projection.Accessor
	currentReceivedAccessor     projection.Accessor
	currentRemotePingTimer      chan timer.Stopper
	currentRelay                relay.Relay
	receivingCloser             chan struct{}
}

// Bootup initialize the request handler
func (p *processor) Bootup() (fsm.State, error) {
	defer p.rw.Done()

	// Read the request header
	_, rErr := io.ReadFull(p.rw, p.cfg.Buffer[:1])

	if rErr != nil {
		return nil, rErr
	}

	receiver, receiverErr := p.registered.Register(projection.ID(
		p.cfg.Buffer[0]))

	if receiverErr != nil {
		rw.WriteFull(p.rw, []byte{RespondJoinErrorNotFound})

		return nil, receiverErr
	}

	p.currentProjectionID = projection.ID(p.cfg.Buffer[0])
	p.currentReceiver = receiver

	// Start dispatcher
	receivingStartErr := p.startReceiving()

	if receivingStartErr != nil {
		p.registered.Remove(p.currentProjectionID)

		rw.WriteFull(p.rw, []byte{RespondJoinErrorInternalFailure})

		return nil, receivingStartErr
	}

	// Tell the client that it's been successfully registered, and a
	// maximum timeout so client knows when send a ping to keepalive
	// the connection
	p.cfg.Buffer[0] = RespondJoined
	p.cfg.Buffer[1] = byte(p.cfg.Timeout >> 4)
	p.cfg.Buffer[2] = byte((p.cfg.Timeout << 4) >> 4)

	_, wErr := rw.WriteFull(p.rw, p.cfg.Buffer[:3])

	if wErr != nil {
		p.stopReceiving()

		p.registered.Remove(p.currentProjectionID)

		return nil, wErr
	}

	return p.wait, nil
}

// receiving waiting a new Accessor to be received and send signal to
// client to start handling procedure
func (p *processor) receiving(l logger.Logger) error {
	for {
		select {
		case accessorReceiver := <-p.currentReceiver.Receive():
			select {
			case p.currentReceivedAccessorChan <- accessorReceiver:
				_, wErr := rw.WriteFull(p.rw, []byte{RequestClientRelayRequest})

				if wErr != nil {
					return wErr
				}

				return nil

			default:
				return ErrProcessorReceivingMulitpleClientsReceived
			}

		case <-p.receivingCloser:
			return nil
		}
	}
}

// startReceiving starts receiving
func (p *processor) startReceiving() error {
	rResult, rJoinErr := p.runner.Run(p.logger, p.receiving, nil)

	if rJoinErr != nil {
		return rJoinErr
	}

	p.currentReceiveResult = rResult

	return nil
}

// stopReceiving stops receiving
func (p *processor) stopReceiving() error {
	if p.currentReceiveResult == nil {
		return ErrProcessorReceivingStopAlreadyClosed
	}

	currentReceiveResult := p.currentReceiveResult
	p.currentReceiveResult = nil

	select {
	case rsl := <-currentReceiveResult:
		return rsl

	case p.receivingCloser <- struct{}{}:
		return <-currentReceiveResult
	}
}

// reinit restarts waiting procedure
func (p *processor) reinit(f fsm.FSM) error {
	sErr := p.startReceiving()

	if sErr != nil {
		return sErr
	}

	f.Switch(p.wait)

	return nil
}

// wait waits request
func (p *processor) wait(f fsm.FSM) error {
	_, rErr := io.ReadFull(p.rw, p.cfg.Buffer[:1])

	if rErr != nil {
		p.rw.Done()

		return rErr
	}

	p.rw.Done()

	switch p.cfg.Buffer[0] {
	case RespondClientRelease:
		select {
		case pingTimer := <-p.currentRemotePingTimer:
			p.currentRemotePingTimer <- pingTimer

			_, wErr := rw.WriteFull(p.rw, []byte{RequestClientReleaseCancel})

			if wErr != nil {
				return wErr
			}

			return nil

		default:
		}

		select {
		case p.currentReceivedAccessor = <-p.currentReceivedAccessorChan:
			p.currentReceivedAccessorChan <- p.currentReceivedAccessor

			_, wErr := rw.WriteFull(p.rw, []byte{RequestClientReleaseCancel})

			if wErr != nil {
				return wErr
			}

			return nil

		default:
		}

		rStopErr := p.stopReceiving()

		switch rStopErr {
		case nil:
		case ErrProcessorReceivingStopAlreadyClosed:

		default:
			_, wErr := rw.WriteFull(p.rw, []byte{RequestClientReleaseCancel})

			if wErr != nil {
				return wErr
			}

			return rStopErr
		}

		// Test again to make sure we haven't received an Accessor
		select {
		case p.currentReceivedAccessor = <-p.currentReceivedAccessorChan:
			p.currentReceivedAccessorChan <- p.currentReceivedAccessor

			_, wErr := rw.WriteFull(p.rw, []byte{RequestClientReleaseCancel})

			if wErr != nil {
				return wErr
			}

			return nil

		default:
		}

		_, wErr := rw.WriteFull(p.rw, []byte{RequestClientRelease})

		if wErr != nil {
			return wErr
		}

		return nil

	case RespondClientQuit:
		return f.Shutdown()

	case RespondClientRelayInitialized:
		select {
		case p.currentReceivedAccessor = <-p.currentReceivedAccessorChan:
			kaStopErr := p.stopReceiving()

			if kaStopErr != nil {
				return kaStopErr
			}

			return f.SwitchTick(p.relayInit)

		default:
			return ErrProcessorWaitUnexpectedClientRelayInitedSignal
		}

	case RespondClientRelayInitializationFailed:
		fallthrough
	case byte(relay.SignalError):
		select {
		case p.currentReceivedAccessor = <-p.currentReceivedAccessorChan:
			kaStopErr := p.stopReceiving()

			if kaStopErr != nil {
				return kaStopErr
			}

			p.currentReceivedAccessor.Error <- ErrProcessorWaitRelayInitFailed

			p.currentReceivedAccessor.Runner = nil
			p.currentReceivedAccessor.Access = nil
			p.currentReceivedAccessor.Error = nil

			return f.SwitchTick(p.reinit)

		default:
			return ErrProcessorWaitUnexpectedClientRelayInitFailureSignal
		}

	case RespondClientPingRequest:
		select {
		case p.currentRemotePingTimer <- p.cfg.ConnectionDelay.Start():
			_, wErr := rw.WriteFull(p.rw, []byte{RequestClientPingEmit})

			if wErr != nil {
				return wErr
			}

			return nil

		default:
			return ErrProcessorWaitNotReadyToHandleClientPingRequest
		}

	case RespondClientPingRespond:
		select {
		case pingTimer := <-p.currentRemotePingTimer:
			pingDelay := pingTimer.Stop()

			p.registered.All(func(receiver projection.Receiver) {
				receiver.Delay(pingDelay)
			})

			return nil

		default:
			return ErrProcessorWaitUnexpectedClientPingRespond
		}

	default:
		return ErrProcessorWaitUnknownRespondReceived
	}
}

// relayInit initialize the relay
func (p *processor) relayInit(f fsm.FSM) error {
	p.currentRelay = relay.New(
		p.logger,
		p.currentReceivedAccessor.Runner,
		p.rw,
		p.cfg.Buffer,
		requestRelay{
			client: p.currentReceivedAccessor.Access,
		}, make([]byte, 4096))

	bootErr := p.currentRelay.Bootup(nil)

	if bootErr != nil {
		p.currentRelay = nil

		p.currentReceivedAccessor.Error <- ErrProcessorAccessorRelayFailed

		p.currentReceivedAccessor.Runner = nil
		p.currentReceivedAccessor.Access = nil
		p.currentReceivedAccessor.Error = nil

		return f.SwitchTick(p.reinit)
	}

	f.Switch(p.relayRelay)

	return nil
}

// relayRelay relays data
func (p *processor) relayRelay(f fsm.FSM) error {
	tkErr := p.currentRelay.Tick()

	if tkErr != nil {
		return tkErr
	}

	if p.currentRelay.Running() {
		return nil
	}

	p.currentRelay = nil

	p.currentReceivedAccessor.Error <- nil

	p.currentReceivedAccessor.Runner = nil
	p.currentReceivedAccessor.Access = nil
	p.currentReceivedAccessor.Error = nil

	return f.SwitchTick(p.reinit)
}

// Shutdown closes current processor
func (p *processor) Shutdown() error {
	p.stopReceiving()

	// Close relay
	if p.currentRelay != nil {
		p.currentRelay.Close()
		p.currentRelay = nil
	}

	// If there're accessors that didn't been picked up by relay, close it
	// here
	select {
	case currentReceivedAccessor := <-p.currentReceivedAccessorChan:
		p.currentReceivedAccessor = currentReceivedAccessor

	default:
	}

	if p.currentReceivedAccessor.Access != nil {
		p.currentReceivedAccessor.Error <- ErrProcessorAccessorClosedForcely

		p.currentReceivedAccessor.Runner = nil
		p.currentReceivedAccessor.Access = nil
		p.currentReceivedAccessor.Error = nil
	}

	p.registered.Remove(p.currentProjectionID)

	return nil
}
