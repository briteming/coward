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
)

// Signal represents a Relay Signal
type Signal byte

// Consts
const (
	SignalData      Signal = 0xFC
	SignalError     Signal = 0xFD
	SignalCompleted Signal = 0xFE
	SignalClose     Signal = 0xFF
)

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
	Client(log logger.Logger, server Server) (io.ReadWriteCloser, error)
}

// relay implements Relay
type relay struct {
	logger                               logger.Logger
	runner                               worker.Runner
	server                               rw.ReadWriteDepleteDoner
	serverBuffer                         []byte
	clientBuilder                        Client
	clientBuffer                         []byte
	serverConnIsDown                     bool
	clientResultChan                     <-chan error
	clientIsDown                         bool
	initiativeClientDownSyncDisabled     bool
	initiativeClientDownSyncDisabledLock sync.Mutex
	clientEnabled                        chan io.ReadWriteCloser
	running                              bool
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
		logger:                               log.Context("Relay"),
		runner:                               runner,
		server:                               server,
		serverBuffer:                         serverBuffer,
		clientBuilder:                        clientBuilder,
		clientBuffer:                         clientBuffer,
		serverConnIsDown:                     false,
		clientResultChan:                     nil,
		clientIsDown:                         false,
		initiativeClientDownSyncDisabled:     false,
		initiativeClientDownSyncDisabledLock: sync.Mutex{},
		clientEnabled:                        make(chan io.ReadWriteCloser, 1),
		running:                              false,
	}
}

// Bootup starts the relay
func (r *relay) Bootup(cancel <-chan struct{}) error {
	if r.running {
		return ErrAlreadyBootedUp
	}

	initErr := r.clientBuilder.Initialize(r.logger, server{
		ReadWriteDepleteDoner: r.server,
	})

	if initErr != nil {
		return initErr
	}

	clientResultChan, runErr := r.runner.Run(r.logger, r.clientReceiver, cancel)

	if runErr == nil {
		r.clientResultChan = clientResultChan
		r.running = true

		return nil
	}

	// If there is any error happened after clientBuilder.Initialized,
	// we have to sync the shutdown so the remote knows to release their
	// end of relay
	_, wErr := rw.WriteFull(r.server, []byte{byte(SignalError)})

	if wErr != nil {
		return wErr
	}

	return runErr
}

// disableInitiativeClientDownSync disables Initiative ClientDown Sync signal
// from been sent and returns whether or not it's current call actually disables
// it (in opponent to it's already been disabled by previous calls)
func (r *relay) disableInitiativeClientDownSync() bool {
	r.initiativeClientDownSyncDisabledLock.Lock()
	defer r.initiativeClientDownSyncDisabledLock.Unlock()

	if r.initiativeClientDownSyncDisabled {
		return false
	}

	r.initiativeClientDownSyncDisabled = true

	return true
}

// clientReceiver receives data from the client and send them to
// remote relay
func (r *relay) clientReceiver(l logger.Logger) error {
	client, clientErr := r.clientBuilder.Client(r.logger, server{
		ReadWriteDepleteDoner: r.server,
	})

	if clientErr != nil {
		return clientErr
	}

	defer client.Close()

	r.clientEnabled <- client

	defer func() {
		<-r.clientEnabled
	}()

	r.clientBuffer[0] = byte(SignalData)

	for {
		rLen, rErr := client.Read(r.clientBuffer[1:])

		if rErr == nil {
			_, wErr := rw.WriteFull(r.server, r.clientBuffer[:rLen+1])

			if wErr != nil {
				return wErr
			}

			continue
		}

		if !r.disableInitiativeClientDownSync() {
			return rErr
		}

		r.clientBuffer[0] = byte(SignalCompleted)

		_, wErr := rw.WriteFull(r.server, r.clientBuffer[:rLen+1])

		if wErr != nil {
			return wErr
		}

		return rErr
	}
}

// serverReceiver receives data from remote relay and send them to
// the client
func (r *relay) serverReceiver(cil io.ReadWriteCloser) error {
	defer r.server.Done()

	for {
		if r.server.Depleted() {
			return nil
		}

		rLen, rErr := r.server.Read(r.serverBuffer[:])

		if rErr != nil {
			r.serverConnIsDown = true

			return rErr
		}

		if cil == nil {
			continue
		}

		rw.WriteFull(cil, r.serverBuffer[:rLen])
	}
}

// Running returns whether or not current Relay is running
func (r *relay) Running() bool {
	return r.running
}

// Tick read data from another relay and deliver them to the client
func (r *relay) Tick() error {
	var cil io.ReadWriteCloser

	// Wait until client is ready, so we can read r.client variable
	// Otherwise it's a nil
	if !r.clientIsDown {
		select {
		case cil = <-r.clientEnabled:
			r.clientEnabled <- cil

		case clientCloseErr := <-r.clientResultChan:
			r.clientIsDown = true

			if clientCloseErr != nil {
				r.logger.Debugf("Relay Client exited early due to "+
					"error: %s", clientCloseErr)
			}
		}
	}

	if r.serverConnIsDown {
		r.server.Done()

		return ErrServerConnIsDown
	}

	_, crErr := io.ReadFull(r.server, r.serverBuffer[:1])

	if crErr != nil {
		r.serverConnIsDown = true
		r.server.Done()

		return crErr
	}

	signal := Signal(r.serverBuffer[0])

	switch signal {
	case SignalData:
		return r.serverReceiver(cil)

	case SignalCompleted: // Remote initialized shutdown and waiting for comfirm
		sendErr := r.serverReceiver(cil)

		if sendErr != nil {
			return sendErr
		}

		if r.disableInitiativeClientDownSync() {
			rCloseErr := r.close(false)

			if rCloseErr != nil {
				return rCloseErr
			}
		}

		_, wErr := rw.WriteFull(r.server, []byte{byte(SignalClose)})

		return wErr

	case SignalError: // Remote relay encountered an error and already closed
		fallthrough
	case SignalClose: // Remote has comfirmed shutdown
		sendErr := r.serverReceiver(cil)

		if sendErr != nil {
			return sendErr
		}

		// Disable InitiativeClientDownSync before close the relay, otherwise
		// we may end up sending SignalCompleted to the remote relay while
		// unable to respond it correctly
		r.disableInitiativeClientDownSync()

		return r.close(false)

	default:
		r.logger.Warningf("Received an unknown Relay Signal \"%d\"", signal)

		return ErrUnknownSignal
	}
}

// release releases current relay
func (r *relay) release(sendInitiativeClientDownSync bool) error {
	needSendCloseSync := false

	if sendInitiativeClientDownSync && r.disableInitiativeClientDownSync() {
		needSendCloseSync = true
	}

	if !r.clientIsDown {
		select {
		case clientCloseErr := <-r.clientResultChan:
			if clientCloseErr != nil {
				r.logger.Debugf("Relay Client has been shutdown with "+
					"error: %s", clientCloseErr)
			}

		case cil := <-r.clientEnabled:
			r.clientEnabled <- cil

			cil.Close()

			<-r.clientResultChan // Should be an EOF error
		}

		r.clientIsDown = true
	}

	close(r.clientEnabled)

	if !needSendCloseSync || r.serverConnIsDown {
		return nil
	}

	_, wErr := rw.WriteFull(r.server, []byte{byte(SignalClose)})

	return wErr
}

// close close current relay
func (r *relay) close(sendInitiativeClientDownSync bool) error {
	if !r.running {
		return ErrClosingInactiveRelay
	}

	r.running = false

	return r.release(sendInitiativeClientDownSync)
}

// Close shutdown the relay
// ALERT: The .Close is not designed to close the relay manually. The reason
// of it been exposed as a public method is we need to provide a way to
// shutdown the relay during an emergency sitcuation (i.e. Shutting down).
// DO NOT call close manually unless it's for droping parent's (r.server)
// connection.
func (r *relay) Close() error {
	return r.close(true)
}
