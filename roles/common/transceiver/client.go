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

package transceiver

import (
	"net"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/timer"
)

// ConnectionID is the consistent ID of a connection
type ConnectionID uint32

// ClientID is the consistent ID of a client
type ClientID uint32

// Destination is the destination of a balance target
type Destination string

// Client represents a Transceiver Client
type Client interface {
	Serve() (Requester, error)
}

// Balancer represents a Transceiver Client Balancer
type Balancer interface {
	Serve() (Balanced, error)
}

// Requester sends requests to a Transceiver Server
type Requester interface {
	ID() ClientID
	Available() bool
	Full() bool
	Request(
		requester net.Addr,
		req RequestBuilder,
		cancel <-chan struct{},
		m Meter) (bool, error)
	Connections() uint32
	Channels() uint32
	Close() error
}

// Balanced sends requests to a automatically selected Transceiver Server
type Balanced interface {
	Clients(r func(ClientID, Requester))
	Size() int
	Request(
		requester net.Addr,
		destName Destination,
		req BalancedRequestBuilder,
		cancel <-chan struct{}) error
	Close() error
}

// RequestBuilder represents a request handler that will be used to
// handle request between Transceiver Client and Server
type RequestBuilder func(
	ConnectionID, rw.ReadWriteDepleteDoner, logger.Logger) fsm.Machine

// BalancedRequestBuilder is the same as the RequestBuilder but for
// Transceiver Client Balancer (Clients)
type BalancedRequestBuilder func(
	ClientID,
	ConnectionID,
	rw.ReadWriteDepleteDoner,
	logger.Logger,
) fsm.Machine

// Meter takes measurement of key delays during requesting
type Meter interface {
	Connection() timer.Stopper
	ConnectionFailure(error)
	Request() timer.Stopper
}
