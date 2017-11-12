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

package request

import (
	"io"
	"net"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/command"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	tcpdial "github.com/reinit/coward/roles/common/network/dialer/tcp"
	"github.com/reinit/coward/roles/common/relay"
)

// TCPIPv6 IPv6 Connent request
type TCPIPv6 struct {
	TCP
}

type tcpIPv6 struct {
	tcp
}

// ID returns current Request ID
func (c TCPIPv6) ID() command.ID {
	return TCPCommandIPv6
}

// New creates a new request context
func (c TCPIPv6) New(rw rw.ReadWriteDepleteDoner) fsm.Machine {
	return &tcpIPv4{
		tcp: tcp{
			rw:                rw,
			buf:               c.Buffer,
			dialTimeout:       c.DialTimeout,
			connectionTimeout: c.ConnectionTimeout,
			runner:            c.Runner,
			relay:             nil,
			cancel:            c.Cancel,
			noLocalAccess:     c.NoLocalAccess,
		},
	}
}

func (c *tcpIPv6) Bootup() (fsm.State, error) {
	_, rErr := io.ReadFull(c.rw, c.buf[:19])

	if rErr != nil {
		c.rw.Done()

		return nil, rErr
	}

	c.rw.Done()

	ipv6 := net.IP{
		c.buf[0], c.buf[1], c.buf[2], c.buf[3], c.buf[4],
		c.buf[5], c.buf[6], c.buf[7], c.buf[8], c.buf[9],
		c.buf[10], c.buf[11], c.buf[12], c.buf[13], c.buf[14], c.buf[15],
	}

	port := uint16(0)
	port |= uint16(c.buf[16])
	port <<= 8
	port |= uint16(c.buf[17])

	timeout := time.Duration(c.buf[18]) * time.Second

	if timeout == 0 || timeout > c.dialTimeout {
		timeout = c.dialTimeout
	}

	c.relay = relay.New(c.runner, c.rw, c.buf, tcpRelay{
		noLocalAccess:     c.noLocalAccess,
		dialTimeout:       c.dialTimeout,
		connectionTimeout: c.connectionTimeout,
		dial: tcpdial.New(
			ipv6.String(), port, timeout, tcpconn.Wrap).Dialer(),
	}, make([]byte, 4096))

	bootErr := c.relay.Bootup(c.cancel)

	if bootErr != nil {
		return nil, bootErr
	}

	return c.tick, nil
}
