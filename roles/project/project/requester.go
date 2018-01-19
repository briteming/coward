//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2018 Rui NI <ranqus@gmail.com>
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

package project

import (
	"errors"
	"io"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/projector/request"
	"github.com/reinit/coward/roles/projector/request/join"
)

// Errors
var (
	ErrHandlerJoinRespondProjectionNotFound = errors.New(
		"Projection is not defined on the Projector server")

	ErrHandlerJoinRespondInternalFailure = errors.New(
		"Projector server has failed to handle Projection " +
			"registeration request")

	ErrHandlerJoinUnknownRespond = errors.New(
		"Projector server returned an unknown respond")

	ErrHandlerKeepAliveTickerClosed = errors.New(
		"KeepAlive Ticker has been closed")

	ErrHandlerKeepAliveStartAlreadyStarted = errors.New(
		"Can't start KeepAlive because it already been started")

	ErrHandlerKeepAliveStopAlreadyStopped = errors.New(
		"Can't stop KeepAlive because it already been stopped")

	ErrHandlerWaitUnknownSignal = errors.New(
		"Unknown Wait signal has been received")

	ErrHandlerWaitUnexpectedPingEmit = errors.New(
		"Unexpected Ping Emit signal")

	ErrHandlerWaitUnexpectedRelease = errors.New(
		"Unexpected Release signal")
)

type requesterKeepAliveTickResume bool

const (
	requesterKeepAliveTickResumePing requesterKeepAliveTickResume = true
	requesterKeepAliveTickResumeQuit requesterKeepAliveTickResume = false
)

type requesterKeepAliveTickResumer struct {
	resumer chan struct{}
	is      requesterKeepAliveTickResume
}

func (r requesterKeepAliveTickResumer) Is(
	is requesterKeepAliveTickResume) bool {
	return r.is == is
}

func (r requesterKeepAliveTickResumer) Resume() {
	if r.resumer == nil {
		return
	}

	r.resumer <- struct{}{}
}

// requester handling the request
type requester struct {
	projection            Endpoint
	dialer                network.Dialer
	runner                worker.Runner
	buf                   []byte
	project               *project
	pingTickTimeout       time.Duration
	idleQuitTimeout       time.Duration
	currentRelay          relay.Relay
	pendingRelay          bool
	keepAliveResult       chan error
	keepAlivePingTicker   chan connectionPingTicker
	keepAlivePeriodTicker <-chan time.Time
	keepAliveTickResume   chan requesterKeepAliveTickResumer
	keepAliveQuit         chan struct{}
	rw                    rw.ReadWriteDepleteDoner
	log                   logger.Logger
	serverPingDelay       time.Duration
	noCloseSignalToRemote bool
}

// Bootup startup requests
func (h *requester) Bootup() (fsm.State, error) {
	defer h.rw.Done()

	_, wErr := rw.WriteFull(h.rw, []byte{
		request.RequestCommandJoin, byte(h.projection.ID)})

	if wErr != nil {
		return nil, wErr
	}

	_, rErr := io.ReadFull(h.rw, h.buf[:1])

	if rErr != nil {
		return nil, rErr
	}

	switch h.buf[0] {
	case join.RespondJoined:
		// Fall out, we will handle it out side this switch

	case join.RespondJoinErrorNotFound:
		return nil, ErrHandlerJoinRespondProjectionNotFound

	case join.RespondJoinErrorInternalFailure:
		return nil, ErrHandlerJoinRespondInternalFailure

	default:
		return nil, ErrHandlerJoinUnknownRespond
	}

	// Read max timeout: We must send at least one ping within this timeout
	// or server will disconnect us
	_, rErr = io.ReadFull(h.rw, h.buf[:2])

	if rErr != nil {
		rw.WriteFull(h.rw, []byte{join.RespondClientQuit})

		return nil, rErr
	}

	pingDelay := uint16(0)
	pingDelay |= uint16(h.buf[0])
	pingDelay <<= 4
	pingDelay |= uint16(h.buf[1])

	h.serverPingDelay = (time.Duration(pingDelay) * time.Second) / 2

	if h.serverPingDelay > h.pingTickTimeout {
		h.serverPingDelay = h.pingTickTimeout
	}

	kaStartErr := h.startKeepalive()

	if kaStartErr != nil {
		rw.WriteFull(h.rw, []byte{join.RespondClientQuit})

		return nil, kaStartErr
	}

	h.log.Infof("Serving. Ping delay %s", h.serverPingDelay)

	return h.wait, nil
}

// keepalive sends ping request to keep the connection alive
func (h *requester) keepalive(l logger.Logger) error {
	pingTickerEnabled := false
	idleQuitTime := time.Now().Add(h.idleQuitTimeout)
	currentConnectionPingTicker := connectionPingTicker{
		Ticker: nil,
		Resume: nil,
		Next:   time.Time{},
	}

	h.project.IncreaseIdleCount()

	defer func() {
		h.project.DecreaseIdleCount()

		if pingTickerEnabled {
			h.keepAlivePingTicker <- currentConnectionPingTicker
		}
	}()

	var currentConnectionPingTickerChan <-chan time.Time

	currentConnectionPeriodTickerChan := h.keepAlivePeriodTicker

	for {
		select {
		case <-h.keepAliveQuit:
			return nil

		case pingTicker, ok := <-h.keepAlivePingTicker:
			if !ok {
				return ErrHandlerKeepAliveTickerClosed
			}

			// When we received a ping ticker, an initial state will be set:
			//
			// First, we declare the pingTicker as enabled, so it will be
			// returned to the h.keepAlivePingTicker channel for other
			// keepalives to receive.
			//
			// Then, we record the received keepAlivePingTicker to
			// currentConnectionPingTicker so we know what we need to return.
			//
			// Last, we apply currentConnectionPingTicker.Ticker to
			// currentConnectionPingTickerChan, so the pingTicker can actually
			// began.

			pingTickerEnabled = true
			currentConnectionPingTicker = pingTicker
			currentConnectionPingTickerChan = currentConnectionPingTicker.Ticker

			now := time.Now()

			if now.Before(pingTicker.Next) {
				continue
			}

			// Trigger a ping send when we received a ping ticker
			select {
			case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
				resumer: currentConnectionPingTicker.Resume,
				is:      requesterKeepAliveTickResumePing,
			}:
				currentConnectionPingTicker.Next = time.Now().Add(
					h.serverPingDelay)

				currentConnectionPingTickerChan = nil
				currentConnectionPeriodTickerChan = nil

				_, wErr := rw.WriteFull(
					h.rw, []byte{join.RespondClientPingRequest})

				if wErr != nil {
					return wErr
				}

			default:
			}

			l.Debugf("Pinger received")

		case <-currentConnectionPingTicker.Resume:
			currentConnectionPingTickerChan = currentConnectionPingTicker.Ticker
			currentConnectionPeriodTickerChan = h.keepAlivePeriodTicker

		case <-currentConnectionPingTickerChan:
			now := time.Now()

			if now.Before(currentConnectionPingTicker.Next) {
				continue
			}

			currentConnectionPingTicker.Next = now.Add(h.serverPingDelay)

			// Make sure we don't send another ping when current ping hasn't
			// been finished
			select {
			case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
				resumer: currentConnectionPingTicker.Resume,
				is:      requesterKeepAliveTickResumePing,
			}:
				currentConnectionPingTickerChan = nil
				currentConnectionPeriodTickerChan = nil

				_, wErr := rw.WriteFull(
					h.rw, []byte{join.RespondClientPingRequest})

				if wErr != nil {
					return wErr
				}

				l.Debugf("Ping requested")

			default:
				// There are another Ping ticker still waiting to be finished
			}

		case <-currentConnectionPeriodTickerChan:
			now := time.Now()

			if now.Before(idleQuitTime) {
				continue
			}

			idleQuitTime = now.Add(h.idleQuitTimeout)

			select {
			case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
				resumer: currentConnectionPingTicker.Resume,
				is:      requesterKeepAliveTickResumeQuit,
			}:
				// Stop both PingTicker and PeriodTicker.
				//
				// Notice They're designed to be re-enabled by writing to the
				// currentConnectionPingTicker.Resume channel, however it's
				// largly unnecessary to actually do because we're quiting :)

				currentConnectionPingTickerChan = nil
				currentConnectionPeriodTickerChan = nil

				// Notice NO ANY other command will be send after Quit request
				_, wErr := rw.WriteFull(h.rw, []byte{join.RespondClientRelease})

				if wErr != nil {
					return wErr
				}

				l.Debugf("Release requested")

			default:
			}
		}
	}
}

// startKeepalive starts keepalive
func (h *requester) startKeepalive() error {
	if h.keepAliveResult != nil {
		return ErrHandlerKeepAliveStartAlreadyStarted
	}

	kaResult, kaStartErr := h.runner.Run(h.log, h.keepalive, nil)

	if kaStartErr != nil {
		return kaStartErr
	}

	h.keepAliveResult = kaResult

	return nil
}

// stopKeepalive stops keepalive
func (h *requester) stopKeepalive() error {
	if h.keepAliveResult == nil {
		return ErrHandlerKeepAliveStopAlreadyStopped
	}

	kaResult := h.keepAliveResult
	h.keepAliveResult = nil

	select {
	case h.keepAliveQuit <- struct{}{}:
		return <-kaResult

	case r := <-kaResult:
		return r
	}
}

// reinit restart the request waiting procedure
func (h *requester) reinit(f fsm.FSM) error {
	kaStartErr := h.startKeepalive()

	if kaStartErr != nil {
		return kaStartErr
	}

	f.Switch(h.wait)

	return nil
}

// wait waits request from server
func (h *requester) wait(f fsm.FSM) error {
	defer h.rw.Done()

	_, rErr := io.ReadFull(h.rw, h.buf[:1])

	if rErr != nil {
		return rErr
	}

	switch h.buf[0] {
	case join.RequestClientRelayRequest:
		// h.keepalivePingResume must be writable during switching,
		// If it's not, a ping/quit may still undergoing
		select {
		case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
			resumer: nil,
			is:      requesterKeepAliveTickResumePing, // Fake
		}: // Test writability
			kaStopErr := h.stopKeepalive()

			// Cancel that test write AFTER we closed the keepalive loop,
			// so no request will be send to remote
			<-h.keepAliveTickResume

			if kaStopErr != nil {
				return kaStopErr
			}

			h.log.Debugf("Executing new Relay request")

			return f.SwitchTick(h.relayInit)

		default:
		}

		h.pendingRelay = true

		h.log.Debugf("Relay request delayed")

		return nil

	case join.RequestClientRelease: // Notice Release is the last remote signal
		h.stopKeepalive()

		select {
		case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
			resumer: nil,
			is:      requesterKeepAliveTickResumePing, // Fake
		}:
			<-h.keepAliveTickResume

		case resumeChan := <-h.keepAliveTickResume:
			if !resumeChan.Is(requesterKeepAliveTickResumeQuit) {
				return ErrHandlerWaitUnexpectedRelease
			}

		default:
		}

		return f.Shutdown()

	case join.RequestClientKill:
		h.noCloseSignalToRemote = true

		return f.Shutdown()

	case join.RequestClientReleaseCancel:
		select {
		case resumeChan := <-h.keepAliveTickResume:
			if !resumeChan.Is(requesterKeepAliveTickResumeQuit) {
				return ErrHandlerWaitUnexpectedRelease
			}

			resumeChan.Resume()

			if !h.pendingRelay {
				h.log.Debugf("Release request has been refused by remote")

				return nil
			}

			kaStopErr := h.stopKeepalive()

			if kaStopErr != nil {
				return kaStopErr
			}

			h.log.Debugf("Executing delayed Relay request")

			return f.SwitchTick(h.relayInit)

		default:
			return ErrHandlerWaitUnexpectedRelease
		}

	case join.RequestClientPingEmit:
		select {
		case resumeChan := <-h.keepAliveTickResume:
			if !resumeChan.Is(requesterKeepAliveTickResumePing) {
				return ErrHandlerWaitUnexpectedPingEmit
			}

			_, wErr := rw.WriteFull(
				h.rw, []byte{join.RespondClientPingRespond})

			if wErr != nil {
				return wErr
			}

			h.log.Debugf("Ping emitted")

			resumeChan.Resume()

			if !h.pendingRelay {
				return nil
			}

			kaStopErr := h.stopKeepalive()

			if kaStopErr != nil {
				return kaStopErr
			}

			h.log.Debugf("Executing delayed Relay request")

			return f.SwitchTick(h.relayInit)

		default:
			return ErrHandlerWaitUnexpectedPingEmit
		}

	default:
		return ErrHandlerWaitUnknownSignal
	}
}

// relayInit initialize relay
func (h *requester) relayInit(f fsm.FSM) error {
	h.pendingRelay = false

	relay := relay.New(h.log, h.runner, h.rw, h.buf, &relaying{
		dial:       h.dialer.Dialer(),
		projection: h.projection,
		client:     nil,
	}, make([]byte, 4096))

	bootErr := relay.Bootup(nil)

	// If that Bootup has failed, it will return an error signal by it self,
	// so we don't need to send anything
	if bootErr != nil {
		h.log.Debugf("Relay bootup failed: %s", bootErr)

		// Continue executing without handle bootErr
		return f.SwitchTick(h.reinit)
	}

	h.currentRelay = relay

	f.Switch(h.relayExchange)

	return nil
}

// relayExchange exchange data between the server and relay
func (h *requester) relayExchange(f fsm.FSM) error {
	tErr := h.currentRelay.Tick()

	if tErr != nil {
		return tErr
	}

	if h.currentRelay.Running() {
		return nil
	}

	h.currentRelay = nil

	h.log.Debugf("Relay request completed")

	return f.SwitchTick(h.reinit)
}

// Shutdown close current handling
func (h *requester) Shutdown() error {
	h.stopKeepalive()

	if h.currentRelay != nil {
		h.currentRelay.Close()
		h.currentRelay = nil
	}

	if !h.noCloseSignalToRemote {
		_, wErr := rw.WriteFull(h.rw, []byte{join.RespondClientQuit})

		if wErr != nil {
			return wErr
		}
	}

	h.log.Debugf("Bye")

	return nil
}
