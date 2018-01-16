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

package request

import (
	"io"
	"net"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
)

type tcpRelay struct {
	noLocalAccess     bool
	dialTimeout       time.Duration
	connectionTimeout time.Duration
	dial              network.Dial
}

func (c tcpRelay) Initialize(l logger.Logger, server relay.Server) error {
	return nil
}

func (c tcpRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	remoteConn, remoteDialErr := c.dial.Dial()

	if remoteDialErr != nil {
		_, wErr := server.Write([]byte{TCPRespondUnreachable})

		if wErr != nil {
			return nil, wErr
		}

		return nil, remoteDialErr
	}

	remoteAddr := remoteConn.RemoteAddr().(*net.TCPAddr)

	if c.noLocalAccess &&
		(remoteAddr.IP.IsLoopback() ||
			remoteAddr.IP.IsUnspecified() ||
			remoteAddr.IP.IsMulticast()) {
		remoteConn.Close()

		_, wErr := server.Write([]byte{TCPRespondAccessDeined})

		if wErr != nil {
			return nil, wErr
		}

		return nil, ErrTCPLocalAccessDeined
	}

	_, wErr := server.Write([]byte{TCPRespondOK})

	if wErr != nil {
		remoteConn.Close()

		return nil, wErr
	}

	remoteConn.SetTimeout(c.connectionTimeout)

	return remoteConn, nil
}
