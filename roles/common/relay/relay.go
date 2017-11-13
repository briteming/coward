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

package relay

import (
	"errors"
	"io"

	"github.com/reinit/coward/common/corunner"
	"github.com/reinit/coward/common/rw"
)

// Errors
var (
	ErrUnknownSignal = errors.New(
		"Unknown Relay Signal")

	ErrAlreadyBootedUp = errors.New(
		"Relay already been booted up")

	ErrClosingInactiveRelay = errors.New(
		"Closing an inactive Relay")
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
	Initialize(server Server) error
	Client(server Server) (io.ReadWriteCloser, error)
}

// relay implements Relay
type relay struct {
	runner                       corunner.Runner
	server                       rw.ReadWriteDepleteDoner
	serverBuffer                 []byte
	clientBuilder                Client
	clientBuffer                 []byte
	client                       io.ReadWriteCloser
	serverConnIsDown             bool
	runResultChan                <-chan error
	runHasDown                   chan struct{}
	suppressClientDownSync       chan struct{}
	initiativeClientDownSync     chan struct{}
	initiativeClientDownSyncSent chan struct{}
	clientEnabled                chan struct{}
	clientSkipWrite              bool
	running                      bool
}

// New creates a new Relay
func New(
	runner corunner.Runner,
	server rw.ReadWriteDepleteDoner,
	serverBuffer []byte,
	clientBuilder Client,
	clientBuffer []byte,
) Relay {
	return &relay{
		runner:                       runner,
		server:                       server,
		serverBuffer:                 serverBuffer,
		clientBuilder:                clientBuilder,
		clientBuffer:                 clientBuffer,
		client:                       nil,
		serverConnIsDown:             false,
		runResultChan:                make(<-chan error),
		runHasDown:                   make(chan struct{}, 1),
		suppressClientDownSync:       make(chan struct{}, 1),
		initiativeClientDownSync:     make(chan struct{}, 1),
		initiativeClientDownSyncSent: make(chan struct{}, 1),
		clientEnabled:                make(chan struct{}, 1),
		clientSkipWrite:              false,
		running:                      false,
	}
}

// Bootup starts the relay
func (r *relay) Bootup(cancel <-chan struct{}) error {
	if r.running {
		return ErrAlreadyBootedUp
	}

	initErr := r.clientBuilder.Initialize(server{
		ReadWriteDepleteDoner: r.server,
	})

	if initErr != nil {
		return initErr
	}

	runResultChan, runErr := r.runner.Run(r.clientReceiver, cancel)

	if runErr == nil {
		r.runResultChan = runResultChan
		r.running = true

		return nil
	}

	// If there is any error happened after clientBuilder.Initialized,
	// we have to sync the shutdown so the remote knows to release their
	// end of relay
	_, wErr := r.server.Write([]byte{byte(SignalError)})

	if wErr != nil {
		return wErr
	}

	return runErr
}

// clientReceiver receives data from the client and send them to
// the another relay
func (r *relay) clientReceiver() error {
	client, clientErr := r.clientBuilder.Client(server{
		ReadWriteDepleteDoner: r.server,
	})

	if clientErr != nil {
		return clientErr
	}

	defer client.Close()

	r.client = client

	r.clientEnabled <- struct{}{}

	r.clientBuffer[0] = byte(SignalData)

	for {
		rLen, rErr := client.Read(r.clientBuffer[1:])

		if rErr == nil {
			_, wErr := r.server.Write(r.clientBuffer[:rLen+1])

			if wErr != nil {
				return wErr
			}

			continue
		}

		select {
		case <-r.suppressClientDownSync:
			return nil

		case r.initiativeClientDownSync <- struct{}{}:
			// Channels are selected randomly, so
			select {
			case <-r.suppressClientDownSync:
				// Cancel initiativeClientDownSync
				<-r.initiativeClientDownSync

				return nil

			default:
			}

			r.clientBuffer[0] = byte(SignalCompleted)

			_, wErr := r.server.Write(r.clientBuffer[:rLen+1])

			// Write it in, don't care about error
			r.initiativeClientDownSyncSent <- struct{}{}

			if wErr != nil {
				return wErr
			}

			return rErr
		}
	}
}

// Running returns whether or not current Relay is running
func (r *relay) Running() bool {
	return r.running
}

// Tick read data from another relay and deliver them to the client
func (r *relay) Tick() error {
	// Wait until client is ready, so we can read r.client variable
	// Otherwise it's a nil
	select {
	case <-r.clientEnabled:
		r.clientEnabled <- struct{}{}

	case <-r.runResultChan:
		r.runHasDown <- struct{}{}

		r.clientSkipWrite = true
	}

	_, crErr := io.ReadFull(r.server, r.serverBuffer[:1])

	if crErr != nil {
		r.server.Done()

		return crErr
	}

	signalID := r.serverBuffer[0]

	for {
		if r.server.Depleted() {
			break
		}

		rLen, rErr := r.server.Read(r.serverBuffer[:])

		if rErr != nil {
			r.serverConnIsDown = true
			r.server.Done()

			return rErr
		}

		if r.clientSkipWrite {
			continue
		}

		rw.WriteFull(r.client, r.serverBuffer[:rLen])
	}

	r.server.Done()

	switch Signal(signalID) {
	case SignalData:
		return nil //tickErr

	case SignalError: // Remote relay encountered an error and already closed
		select {
		case r.initiativeClientDownSync <- struct{}{}:
		default:
		}

		select {
		case r.suppressClientDownSync <- struct{}{}:
		default:
		}

		return r.Close()

	case SignalCompleted: // Remote initialized shutdown
		// If we are shutting down too, ignore the incomming
		// SignalCompleted
		select {
		case <-r.initiativeClientDownSync:
			r.initiativeClientDownSync <- struct{}{}

			<-r.initiativeClientDownSyncSent
			r.initiativeClientDownSyncSent <- struct{}{}

			// Must send it AFTER SignalCompleted is sent to
			// make sure the opponent receives SignalCompleted
			// before SignalClose
			_, wErr := r.server.Write([]byte{byte(SignalClose)})

			if wErr != nil {
				return wErr
			}

			return nil

		default:
		}

		fallthrough

	case SignalClose: // Remote has comfirmed shutdown
		select {
		case r.suppressClientDownSync <- struct{}{}:
		default:
		}

		return r.Close()

	default:
		return ErrUnknownSignal
	}
}

func (r *relay) close() error {
	select {
	case <-r.runHasDown:
		r.runHasDown <- struct{}{}
	case <-r.runResultChan:

	case <-r.clientEnabled:
		r.clientEnabled <- struct{}{}

		r.client.Close()

		select {
		case <-r.runHasDown:
			r.runHasDown <- struct{}{}
		case <-r.runResultChan:
		}
	}

	closeInitializedBySelf := false

	select {
	case <-r.initiativeClientDownSync:
		closeInitializedBySelf = true

	default:
	}

	if closeInitializedBySelf || r.serverConnIsDown {
		return nil
	}

	_, wErr := r.server.Write([]byte{byte(SignalClose)})

	if wErr != nil {
		return wErr
	}

	return nil
}

// Close shutdown the relay
func (r *relay) Close() error {
	if !r.running {
		return ErrClosingInactiveRelay
	}

	r.running = false

	return r.close()
}
