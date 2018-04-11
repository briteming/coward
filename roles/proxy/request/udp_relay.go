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
	"sync"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network/resolve"
	"github.com/reinit/coward/roles/common/relay"
)

type udpRelay struct {
	localAddr net.Addr
	listenIP  net.IP
}

func (u *udpRelay) Initialize(l logger.Logger, server relay.Server) error {
	spLocalHost, _, spLocalErr := net.SplitHostPort(u.localAddr.String())

	if spLocalErr != nil {
		rw.WriteFull(server, []byte{UDPRespondInvalidRequest})

		return ErrUDPTransportFailedToGetLocalIP
	}

	localIP := net.ParseIP(spLocalHost)

	if localIP == nil {
		rw.WriteFull(server, []byte{UDPRespondInvalidRequest})

		return ErrUDPTransportFailedToGetLocalIP
	}

	u.listenIP = localIP

	return nil
}

func (u *udpRelay) Abort(l logger.Logger, aborter relay.Aborter) error {
	return aborter.SendError()
}

func (u *udpRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	listener, listenErr := net.ListenUDP("udp", nil)

	if listenErr != nil {
		rw.WriteFull(server, []byte{UDPRespondFailedToListen})

		return nil, listenErr
	}

	listenerConn := &udpConn{
		UDPConn:    listener,
		resolver:   resolve.Cached(1*time.Hour, 10*time.Second, 16),
		remotes:    make(map[resolve.IPMark]struct{}, 16),
		maxRemotes: 16,
		remoteLock: sync.RWMutex{},
	}

	_, wErr := rw.WriteFull(server, []byte{UDPRespondOK})

	if wErr != nil {
		listenerConn.Close()

		return nil, wErr
	}

	return listenerConn, nil
}
