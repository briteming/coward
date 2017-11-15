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
	"bytes"
	"errors"
	"io"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/command"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	tcpdial "github.com/reinit/coward/roles/common/network/dialer/tcp"
	"github.com/reinit/coward/roles/common/relay"
)

// Errors
var (
	ErrTCPHostAddressWasTooLong = errors.New(
		"Failed to read Host Address of a TCP request, data was too long")
)

// TCPHost Connect request
type TCPHost struct {
	TCP
}

type tcpHost struct {
	tcp
}

// ID returns current Request ID
func (c TCPHost) ID() command.ID {
	return TCPCommandHost
}

// New creates a new context
func (c TCPHost) New(rw rw.ReadWriteDepleteDoner) fsm.Machine {
	return &tcpHost{
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

func (c *tcpHost) Bootup() (fsm.State, error) {
	_, rErr := io.ReadFull(c.rw, c.buf[:1])

	if rErr != nil {
		c.rw.Done()

		return nil, rErr
	}

	var rLen int

	rLen, rErr = io.ReadFull(c.rw, c.buf[:c.buf[0]+3])

	if rErr != nil {
		c.rw.Done()

		return nil, rErr
	}

	c.rw.Done()

	host := bytes.TrimSpace(c.buf[:rLen-3])

	port := uint16(0)
	port |= uint16(c.buf[rLen-3])
	port <<= 8
	port |= uint16(c.buf[rLen-2])

	timeout := time.Duration(c.buf[rLen-1]) * time.Second

	if timeout <= 0 {
		c.rw.Write([]byte{TCPRespondBadRequest})

		return nil, ErrTCPInvalidTimeout
	}

	if timeout > c.dialTimeout {
		timeout = c.dialTimeout
	}

	c.relay = relay.New(c.runner, c.rw, c.buf, tcpRelay{
		noLocalAccess:     c.noLocalAccess,
		dialTimeout:       c.dialTimeout,
		connectionTimeout: c.connectionTimeout,
		dial: tcpdial.New(
			string(host), port, timeout, tcpconn.Wrap).Dialer(),
	}, make([]byte, 4096))

	bootErr := c.relay.Bootup(c.cancel)

	if bootErr != nil {
		return nil, bootErr
	}

	return c.tick, nil
}
