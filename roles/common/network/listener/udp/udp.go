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

package udp

import (
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/reinit/coward/roles/common/network"
)

const (
	deadlineTickDelay = 300 * time.Millisecond
)

type listener struct {
	host              net.IP
	port              uint16
	timeout           time.Duration
	maxSize           uint32
	buffer            []byte
	connectionWrapper network.ConnectionWrapper
}

// New creates a new UDP Listener
func New(
	host net.IP,
	port uint16,
	timeout time.Duration,
	maxSize uint32,
	buf []byte,
	connectionWrapper network.ConnectionWrapper,
) network.Listener {
	return listener{
		host:              host,
		port:              port,
		timeout:           timeout,
		maxSize:           maxSize,
		buffer:            buf,
		connectionWrapper: connectionWrapper,
	}
}

// Listen start listening
func (t listener) Listen() (network.Acceptor, error) {
	listen, listenErr := net.ListenUDP("udp", &net.UDPAddr{
		IP:   t.host,
		Port: int(t.port),
		Zone: "",
	})

	if listenErr != nil {
		return nil, listenErr
	}

	return &acceptor{
		listener:          listen,
		timeout:           t.timeout,
		deadlineTick:      time.NewTicker(deadlineTickDelay),
		maxSize:           t.maxSize,
		buffer:            t.buffer,
		connectionWrapper: t.connectionWrapper,
		clients: clients{
			clients: make(map[network.ConnectionID]*conn, t.maxSize),
			lock:    sync.Mutex{},
			maxSize: t.maxSize,
		},
		rbufferCompleted: make(chan struct{}, 1),
		closed:           make(chan struct{}, 1),
	}, nil
}

// String returns configured address of current listener (Not actual address)
func (t listener) String() string {
	return net.JoinHostPort(
		t.host.String(), strconv.FormatUint(uint64(t.port), 10))
}
