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

package tcp

import (
	"net"
	"strconv"

	"github.com/reinit/coward/roles/common/network"
)

// listener is a TCP listener
type listener struct {
	host              net.IP
	port              uint16
	connectionWrapper network.ConnectionWrapper
}

// acceptor is a TCP acceptor
type acceptor struct {
	listener          *net.TCPListener
	connectionWrapper network.ConnectionWrapper
	closed            chan struct{}
}

// New creates a new TCP listener
func New(
	host net.IP,
	port uint16,
	connectionWrapper network.ConnectionWrapper,
) network.Listener {
	return listener{
		host:              host,
		port:              port,
		connectionWrapper: connectionWrapper,
	}
}

// Listen listens a TCP port
func (t listener) Listen() (network.Acceptor, error) {
	listener, listenErr := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   t.host,
		Port: int(t.port), // Safe when not running on a system that below 16b
		Zone: "",
	})

	if listenErr != nil {
		return nil, listenErr
	}

	return acceptor{
		listener:          listener,
		connectionWrapper: t.connectionWrapper,
		closed:            make(chan struct{}),
	}, nil
}

// String returns current Listener information in string
func (t listener) String() string {
	return net.JoinHostPort(
		t.host.String(), strconv.FormatUint(uint64(t.port), 10))
}

// Addr returns the current address this listener is listen on
func (a acceptor) Addr() net.Addr {
	return a.listener.Addr()
}

// Accept accepts a TCP connection
func (a acceptor) Accept() (network.Connection, error) {
	accepted, acceptErr := a.listener.AcceptTCP()

	if acceptErr != nil {
		return nil, acceptErr
	}

	optErr := accepted.SetLinger(0)

	if optErr != nil {
		return nil, optErr
	}

	// Delay data for sending on server, so the TCP can work more efficienly
	optErr = accepted.SetNoDelay(false)

	if optErr != nil {
		return nil, optErr
	}

	return a.connectionWrapper(accepted), nil
}

// Closed return whether or not current acceptor is closed
func (a acceptor) Closed() chan struct{} {
	return a.closed
}

// Close closes the TCP listener
func (a acceptor) Close() error {
	close(a.closed)

	return a.listener.Close()
}
