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

package udp

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/reinit/coward/common/ticker"
	"github.com/reinit/coward/roles/common/network"
)

// Errors
var (
	ErrTooManyClients = errors.New(
		"Too many UDP clients")
)

type acceptor struct {
	listener          *net.UDPConn
	timeout           time.Duration
	deadlineTick      ticker.Requester
	maxSize           uint32
	buffer            []byte
	connectionWrapper network.ConnectionWrapper
	clients           clients
	rbufferCompleted  chan struct{}
	closed            chan struct{}
}

func (a *acceptor) Addr() net.Addr {
	return a.listener.LocalAddr()
}

func (a *acceptor) Accept() (network.Connection, error) {
	var selectedConn network.Connection

	for selectedConn == nil {
		select {
		case <-a.closed:
			close(a.rbufferCompleted)

			return nil, io.EOF

		case a.rbufferCompleted <- struct{}{}:
		}

		rLen, rAddr, rErr := a.listener.ReadFromUDP(a.buffer)

		if rErr != nil {
			return nil, rErr
		}

		if a.clients.Size() > a.maxSize {
			return nil, ErrTooManyClients
		}

		ipPort := network.ConnectionID(
			a.listener.LocalAddr().String() + ":" + rAddr.String())

		a.clients.Fetch(ipPort, func(cc *conn) {
			select {
			case cc.readerDeliver <- &rbuffer{
				buf:       a.buffer[:rLen],
				remain:    rLen,
				completed: a.rbufferCompleted,
			}:

			default:
				// Drop the datagram
				<-a.rbufferCompleted
			}
		}, func() *conn {
			newConn := &conn{
				conn:                a.listener,
				addr:                rAddr,
				clients:             &a.clients,
				deadlineTicker:      a.deadlineTick,
				ipPort:              ipPort,
				currentReader:       nil,
				readerDeliver:       make(chan *rbuffer, 1),
				readDeadline:        time.Time{},
				readDeadlineEnabled: false,
				closed:              false,
			}

			selectedConn = a.connectionWrapper(newConn)

			return newConn
		})
	}

	return selectedConn, nil
}

// Closed return whether or not current acceptor is closed
func (a *acceptor) Closed() chan struct{} {
	return a.closed
}

// Close closes the UDP listener
func (a *acceptor) Close() error {
	cErr := a.listener.Close()

	if cErr != nil {
		return cErr
	}

	close(a.closed)

	a.clients.Clear(func(cc *conn) {
		cc.Kick()
	})

	return nil
}
