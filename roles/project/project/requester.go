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
	"github.com/reinit/coward/common/ticker"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/common/transceiver"
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

	ErrHandlerKeepAlivePingRespondTimedout = errors.New(
		"Server didn't respond the last ping request in given time, " +
			"the connection may be already down")

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
	projection              Endpoint
	dialer                  network.Dialer
	runner                  worker.Runner
	buf                     []byte
	project                 *project
	pingTickTimeout         time.Duration
	connRequestTimeout      time.Duration
	idleQuitTimeout         time.Duration
	currentRelay            relay.Relay
	pendingRelay            bool
	ticker                  ticker.Requester
	noRelease               bool
	keepAliveResult         chan error
	keepAlivePingTickPermit chan connectionPingTickPermit
	keepAliveTickResume     chan requesterKeepAliveTickResumer
	keepAliveQuit           chan struct{}
	rw                      rw.ReadWriteDepleteDoner
	connCtl                 transceiver.ConnectionControl
	log                     logger.Logger
	serverPingDelay         time.Duration
	noCloseSignalToRemote   bool
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
		h.log.Debugf("Server responded with an unknown Join result: %d",
			h.buf[0])

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
	var nextKeepAliveExpire time.Time
	var releaseWaiter ticker.Waiter
	var pingWaiter ticker.Waiter

	pingTickPermitted := false
	currentPingPermit := connectionPingTickPermit{
		Resume: nil,
		Next:   time.Time{},
	}

	h.project.IncreaseIdleCount()

	defer func() {
		h.project.DecreaseIdleCount()

		if pingTickPermitted {
			h.keepAlivePingTickPermit <- currentPingPermit
		}

		if releaseWaiter != nil {
			releaseWaiter.Close()
		}

		if pingWaiter != nil {
			pingWaiter.Close()
		}
	}()

	var pingSignalChan ticker.Wait
	var releaseSignalChan ticker.Wait
	var currentReleaseSignalChan ticker.Wait

	if !h.noRelease {
		waitReq, waitReqErr := h.ticker.Request(
			time.Now().Add(h.idleQuitTimeout))

		if waitReqErr != nil {
			return waitReqErr
		}

		releaseWaiter = waitReq
		releaseSignalChan = waitReq.Wait()

		currentReleaseSignalChan = releaseSignalChan
	}

	for {
		select {
		case <-h.keepAliveQuit:
			return nil

		case pingTickPermit, ok := <-h.keepAlivePingTickPermit:
			if !ok {
				return ErrHandlerKeepAliveTickerClosed
			}

			now := time.Now()

			nextKeepAliveExpire = now.Add(h.connRequestTimeout)
			currentPingPermit = pingTickPermit
			pingTickPermitted = true

			if pingWaiter != nil {
				pingWaiter.Close()
				pingWaiter = nil
			}

			l.Debugf("Pinger received")

			waitReq, waitReqErr := h.ticker.Request(currentPingPermit.Next)

			if waitReqErr != nil {
				return waitReqErr
			}

			pingWaiter = waitReq
			pingSignalChan = waitReq.Wait()

			select {
			case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
				resumer: currentPingPermit.Resume,
				is:      requesterKeepAliveTickResumePing,
			}:
				currentPingPermit.Next = now.Add(h.serverPingDelay)

				currentReleaseSignalChan = nil

				_, wErr := rw.WriteFull(
					h.rw, []byte{join.RespondClientPingRequest})

				if wErr != nil {
					return wErr
				}

				l.Debugf("Ping requested")

			default:
				// There are another Ping ticker still waiting to be finished
			}

		case <-currentPingPermit.Resume:
			currentReleaseSignalChan = releaseSignalChan

		case <-pingSignalChan:
			pingWaiter = nil

			now := time.Now()
			nextPing := now.Add(h.serverPingDelay)

			waitReq, waitReqErr := h.ticker.Request(nextPing)

			if waitReqErr != nil {
				return waitReqErr
			}

			pingWaiter = waitReq
			pingSignalChan = waitReq.Wait()

			// Make sure we don't send another ping when current ping hasn't
			// been finished
			select {
			case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
				resumer: currentPingPermit.Resume,
				is:      requesterKeepAliveTickResumePing,
			}:
				currentPingPermit.Next = nextPing
				nextKeepAliveExpire = now.Add(h.connRequestTimeout)

				currentReleaseSignalChan = nil

				_, wErr := rw.WriteFull(
					h.rw, []byte{join.RespondClientPingRequest})

				if wErr != nil {
					return wErr
				}

				l.Debugf("Ping requested")

			default:
				// Check whether or not we are excceed the max respond wait
				// time and kill the connection if it did.
				if now.After(nextKeepAliveExpire) {
					h.connCtl.Demolish()

					l.Debugf("Server failed to respond the last Ping request " +
						"in time, connection maybe already lost")

					return ErrHandlerKeepAlivePingRespondTimedout
				}

				// There are another Ping ticker still waiting to be finished
			}

		case <-currentReleaseSignalChan:
			releaseWaiter = nil

			now := time.Now()
			waitReq, waitReqErr := h.ticker.Request(now.Add(h.idleQuitTimeout))

			if waitReqErr != nil {
				return waitReqErr
			}

			releaseWaiter = waitReq
			releaseSignalChan = waitReq.Wait()

			select {
			case h.keepAliveTickResume <- requesterKeepAliveTickResumer{
				resumer: currentPingPermit.Resume,
				is:      requesterKeepAliveTickResumeQuit,
			}:
				nextKeepAliveExpire = now.Add(h.connRequestTimeout)

				currentReleaseSignalChan = nil

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
		h.log.Debugf("Received an unknown request: %d", h.buf[0])

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
