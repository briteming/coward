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
	"github.com/reinit/coward/roles/common/network/resolve"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/proxy/common"
)

type udpMappingRelay struct {
	localAddr      net.Addr
	resolveTimeout time.Duration
	mapped         *common.Mapped
	listenIP       net.IP
}

func (u *udpMappingRelay) Initialize(
	l logger.Logger, server relay.Server) error {
	spLocalHost, _, spLocalErr := net.SplitHostPort(u.localAddr.String())

	if spLocalErr != nil {
		server.Write([]byte{UDPRespondInvalidRequest})

		return ErrUDPTransportFailedToGetLocalIP
	}

	localIP := net.ParseIP(spLocalHost)

	if localIP == nil {
		server.Write([]byte{UDPRespondInvalidRequest})

		return ErrUDPTransportFailedToGetLocalIP
	}

	u.listenIP = localIP

	return nil
}

func (u *udpMappingRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	resolved, resolveErr := resolve.DNS(u.resolveTimeout).Resolve(u.mapped.Host)

	if resolveErr != nil {
		server.Write([]byte{UDPRespondMappingHostUnresolved})

		return nil, resolveErr
	}

	listener, listenErr := net.DialUDP("udp", &net.UDPAddr{
		IP:   u.listenIP,
		Port: 0,
		Zone: "",
	}, &net.UDPAddr{
		IP:   resolved[0],
		Port: int(u.mapped.Port),
		Zone: "",
	})

	if listenErr != nil {
		server.Write([]byte{UDPRespondFailedToListen})

		return nil, listenErr
	}

	_, wErr := server.Write([]byte{UDPRespondOK})

	if wErr != nil {
		listener.Close()

		return nil, wErr
	}

	return listener, nil
}
