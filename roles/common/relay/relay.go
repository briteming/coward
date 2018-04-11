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

package relay

import (
	"errors"
	"io"
	"sync"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/worker"
)

// Errors
var (
	ErrUnknownSignal = errors.New(
		"Unknown Relay Signal")

	ErrAlreadyBootedUp = errors.New(
		"Relay already been booted up")

	ErrClosingInactiveRelay = errors.New(
		"Closing an inactive Relay")

	ErrServerConnIsDown = errors.New(
		"Relay connection is closed")

	ErrRelayNotReady = errors.New(
		"Relay is not ready")
)

// csdStatus Initiative Client Shutdown Sync Status
type csdStatus uint8

// Consts for csdStatus
const (
	csdsUninitialized csdStatus = 0x00
	csdsEnabled       csdStatus = 0x01
	csdsDisabled      csdStatus = 0x02
)

// csdDisabledBy Initiative ClientDown Disable Result
type csdDisabledBy uint8

// Consts for csdDisabledBy
const (
	csddrByMe     csdDisabledBy = 0x00
	csddrByOther  csdDisabledBy = 0x01
	csddrNotReady csdDisabledBy = 0x02
)

// Ticker Relay ticker
type Ticker func() error

// Relay will exchange data between another relay
type Relay interface {
	Bootup(cancel <-chan struct{}) error
	Tick() error
	Running() bool
	Close() error
}

// Client is the data inputer
type Client interface {
	Initialize(log logger.Logger, server Server) error
	Abort(log logger.Logger, aborter Aborter) error
	Client(log logger.Logger, server Server) (io.ReadWriteCloser, error)
}

// relay implements Relay
type relay struct {
	logger           logger.Logger
	runner           worker.Runner
	mode             Ticker
	server           rw.ReadWriteDepleteDoner
	serverBuffer     []byte
	clientBuilder    Client
	clientBuffer     []byte
	serverConnIsDown bool
	clientResultChan <-chan error
	clientIsDown     bool
	csdStatus        csdStatus
	csdStatusLock    sync.Mutex
	clientEnabled    chan io.ReadWriteCloser
	closeSent        bool
	completed        bool
}

// New creates a new Relay
func New(
	log logger.Logger,
	runner worker.Runner,
	server rw.ReadWriteDepleteDoner,
	serverBuffer []byte,
	clientBuilder Client,
	clientBuffer []byte,
) Relay {
	return &relay{
		logger:           log.Context("Relay"),
		runner:           runner,
		mode:             nil,
		server:           server,
		serverBuffer:     serverBuffer,
		clientBuilder:    clientBuilder,
		clientBuffer:     clientBuffer,
		serverConnIsDown: false,
		clientResultChan: nil,
		clientIsDown:     false,
		csdStatus:        csdsUninitialized,
		csdStatusLock:    sync.Mutex{},
		clientEnabled:    make(chan io.ReadWriteCloser, 1),
		closeSent:        false,
		completed:        false,
	}
}

// Bootup starts the relay
func (r *relay) Bootup(cancel <-chan struct{}) error {
	if r.Running() {
		return ErrAlreadyBootedUp
	}

	initErr := r.clientBuilder.Initialize(r.logger, server{
		ReadWriteDepleteDoner: r.server,
		log: r.logger.Context("Client").Context("Initializer"),
	})

	if initErr != nil {
		return initErr
	}

	clientResultChan, runErr := r.runner.Run(
		r.logger, r.clientReceiver, cancel)

	if runErr == nil {
		r.clientResultChan = clientResultChan
		r.mode = r.exchanging

		return nil
	}

	// If there is any error happened after clientBuilder.Initialized,
	// we need to Abort the bootup
	aErr := r.clientBuilder.Abort(r.logger, aborter{
		server: server{
			ReadWriteDepleteDoner: r.server,
			log: r.logger.Context("Client").
				Context("Initializer").Context("Abort"),
		},
		relay: r,
	})

	if aErr != nil {
		return aErr
	}

	return runErr
}

// sendClose sends close signal to close the opponent relay
func (r *relay) sendClose() error {
	if r.closeSent {
		return nil
	}

	r.closeSent = true

	_, wErr := rw.WriteFull(r.server, []byte{byte(SignalClose)})

	return wErr
}

// sendClosed sends close signal to tell opponent relay that current
// relay is closed
func (r *relay) sendClosed() error {
	if r.closeSent {
		return nil
	}

	r.closeSent = true

	_, wErr := rw.WriteFull(r.server, []byte{byte(SignalClosed)})

	return wErr
}

// sendError sends close signal to tell opponent relay that current
// encountered some error and already closed
func (r *relay) sendError() error {
	if r.closeSent {
		return nil
	}

	r.closeSent = true

	_, wErr := rw.WriteFull(r.server, []byte{byte(SignalError)})

	return wErr
}

// closeClient closes client data transmission
func (r *relay) closeClient() error {
	if r.clientIsDown {
		return nil
	}

	var clientCloseErr error

	select {
	case clientCloseErr = <-r.clientResultChan:

	case cil := <-r.clientEnabled:
		r.clientEnabled <- cil

		cil.Close()

		clientCloseErr = <-r.clientResultChan // Should be an EOF error
	}

	r.clientIsDown = true

	close(r.clientEnabled)

	if clientCloseErr != nil {
		r.logger.Debugf("Client has been shutdown: %s", clientCloseErr)
	} else {
		r.logger.Debugf("Client has been shutdown")
	}

	return clientCloseErr
}

// enablecompleteSignalDispatch enables Initiative ClientDown Sync signal.
// It should only be called during initialization
func (r *relay) setCompleteSignalDispatch(s csdStatus) {
	r.csdStatusLock.Lock()
	defer r.csdStatusLock.Unlock()

	r.csdStatus = s
}

// disableCompleteSignalDispatch disables Initiative ClientDown Sync signal
// from been sent and returns whether or not it's current call actually disables
// it (in opponent to it's already been disabled by previous calls)
func (r *relay) disableCompleteSignalDispatch(
	c func(csdDisabledBy) error) error {
	r.csdStatusLock.Lock()
	defer r.csdStatusLock.Unlock()

	switch r.csdStatus {
	case csdsDisabled:
		return c(csddrByOther)

	case csdsEnabled:
		r.csdStatus = csdsDisabled

		return c(csddrByMe)

	default:
		return c(csddrNotReady)
	}
}

// clientReceiver receives data from the client and send them to
// remote relay
func (r *relay) clientReceiver(l logger.Logger) error {
	var rLen int
	var rErr error

	client, clientErr := r.clientBuilder.Client(r.logger, server{
		ReadWriteDepleteDoner: r.server,
		log: r.logger.Context("Client"),
	})

	if clientErr != nil {
		return clientErr
	}

	defer client.Close()

	r.setCompleteSignalDispatch(csdsEnabled)

	r.clientEnabled <- client

	defer func() {
		<-r.clientEnabled

		r.disableCompleteSignalDispatch(func(dr csdDisabledBy) error {
			switch dr {
			case csddrByMe:
				r.clientBuffer[0] = byte(SignalCompleted)

				_, wErr := rw.WriteFull(r.server, r.clientBuffer[:rLen+1])

				return wErr

			case csddrByOther:
				return nil
			}

			return ErrRelayNotReady
		})
	}()

	for {
		rLen, rErr = client.Read(r.clientBuffer[1:])

		if rErr == nil {
			// Must reset it everytime, because sometime the Writer will overwrite
			// the input byte slice
			r.clientBuffer[0] = byte(SignalData)

			_, wErr := rw.WriteFull(r.server, r.clientBuffer[:rLen+1])

			if wErr != nil {
				return wErr
			}

			continue
		}

		return rErr
	}
}

// serverReceiver receives data from remote relay and send them to
// the client
func (r *relay) serverReceiver(cil io.ReadWriteCloser) error {
	if cil == nil {
		return nil
	}

	for {
		if r.server.Depleted() {
			return nil
		}

		rLen, rErr := r.server.Read(r.serverBuffer[:])

		if rErr != nil {
			r.serverConnIsDown = true

			return rErr
		}

		rw.WriteFull(cil, r.serverBuffer[:rLen])
	}
}

// Running returns whether or not current Relay is running
func (r *relay) Running() bool {
	return r.mode != nil
}

// Tick handles Relay tick
func (r *relay) Tick() error {
	if !r.Running() {
		return ErrRelayNotReady
	}

	return r.mode()
}

// clientConn return the Conn of client
func (r *relay) clientConn() io.ReadWriteCloser {
	if r.clientIsDown {
		return nil
	}

	select {
	case cil := <-r.clientEnabled:
		r.clientEnabled <- cil

		return cil

	case clientCloseErr := <-r.clientResultChan:
		r.clientIsDown = true

		if clientCloseErr != nil {
			r.logger.Debugf("Client is actively closed: %s", clientCloseErr)
		} else {
			r.logger.Debugf("Client is actively closed")
		}
	}

	return nil
}

// readSignal reads Relay signal
func (r *relay) readSignal(buf []byte) (Signal, error) {
	if r.serverConnIsDown {
		return 0, ErrServerConnIsDown
	}

	_, crErr := io.ReadFull(r.server, buf[:1])

	if crErr != nil {
		r.serverConnIsDown = true

		return 0, crErr
	}

	return Signal(buf[0]), nil
}

// handleShutdown handles Relay shutdown requests
func (r *relay) handleShutdown(s Signal, cli io.ReadWriteCloser) (bool, error) {
	switch s {
	case SignalCompleted: // Remote initialized shutdown and waiting for comfirm
		if r.completed {
			return false, nil
		}

		r.completed = true

		sendErr := r.serverReceiver(cli)

		if sendErr != nil {
			return true, sendErr
		}

		// Try prevent local SignalCompleted from been sent
		skipClose := false
		dErr := r.disableCompleteSignalDispatch(func(d csdDisabledBy) error {
			switch d {
			case csddrNotReady:
				fallthrough
			case csddrByMe:
				// Fall out and close relay
				return nil

			case csddrByOther:
				// Fallout and do nothing (Waiting for SignalClose, and don't
				// send it)
				skipClose = true

				return r.sendClose()
			}

			// Throw out error, so relay can be closed (And send SignalClose)
			return ErrRelayNotReady
		})

		if dErr != nil {
			return true, dErr
		}

		if skipClose {
			r.logger.Debugf("Client is closed, Relay is waiting to be closed")

			return true, nil
		}

		r.logger.Debugf("Closing, and asking remote Relay to shut down")

		return true, r.close()

	case SignalClosed: // Remote relay already closed
		fallthrough
	case SignalError: // Remote relay encountered an error and already closed
		r.closeSent = true
		fallthrough
	case SignalClose: // Remote has comfirmed shutdown
		r.logger.Debugf("Closing in response to \"%s\" signal", s)

		sendErr := r.serverReceiver(cli)

		if sendErr != nil {
			return true, sendErr
		}

		return true, r.close()
	}

	return false, nil
}

// quiting Relay quiting mode which only accept shutdown requests
func (r *relay) quiting() error {
	defer r.server.Done()

	cli := r.clientConn()

	signal, signalReadErr := r.readSignal(r.serverBuffer[:])

	if signalReadErr != nil {
		return signalReadErr
	}

	matched, downErr := r.handleShutdown(signal, cli)

	if downErr != nil {
		return downErr
	}

	if !matched {
		r.logger.Warningf("Received an unknown Relay Signal \"%s\" "+
			"during quiting", signal)

		return ErrUnknownSignal
	}

	return nil
}

// exchanging read data from another relay and deliver them to the client
func (r *relay) exchanging() error {
	defer r.server.Done()

	cli := r.clientConn()

	s, signalReadErr := r.readSignal(r.serverBuffer[:])

	if signalReadErr != nil {
		return signalReadErr
	}

	// the Only expected signal is SignalData
	if s == SignalData {
		return r.serverReceiver(cli)
	}

	// If signal not expected, set mode to quiting and handle shutdown requests
	r.mode = r.quiting

	matched, downErr := r.handleShutdown(s, cli)

	if downErr != nil {
		return downErr
	}

	if !matched {
		r.logger.Warningf("Received an unknown Relay Signal \"%s\"", s)

		return ErrUnknownSignal
	}

	return nil
}

// close close current relay
func (r *relay) close() error {
	if !r.Running() {
		return ErrClosingInactiveRelay
	}

	r.mode = nil

	// We don't care about error happened during client closing, because
	// client will be closed regardless whether or not there is an error
	// and we can't let error blocks release progress
	r.closeClient()

	return r.sendClosed()
}

// Close shutdown the relay
// ALERT: The .Close is not designed to close the relay manually. The reason
// of it been exposed as a public method is we need to provide a way to
// shutdown the relay during an emergency situation (i.e. Shutting down).
// DO NOT call close manually unless it's for droping parent's (r.server)
// connection.
func (r *relay) Close() error {
	cErr := r.close()

	if cErr != nil {
		return cErr
	}

	r.logger.Debugf("Forcely closed")

	return nil
}
