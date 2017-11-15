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
	"errors"
	"io"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/network"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	tcpdial "github.com/reinit/coward/roles/common/network/dialer/tcp"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/proxy/common"
)

// Errors
var (
	ErrTCPMappingNotFound = errors.New(
		"TCP Mapping not found")
)

// TCPMapping TCP mapping request
type TCPMapping struct {
	TCP

	Mapping common.Mapping
}

type tcpMapping struct {
	tcp

	mapping common.Mapping
}

// ID returns current Request ID
func (c TCPMapping) ID() command.ID {
	return TCPCommandMapping
}

// New creates a new request context
func (c TCPMapping) New(rw rw.ReadWriteDepleteDoner) fsm.Machine {
	return &tcpMapping{
		tcp: tcp{
			logger:            c.Logger,
			buf:               c.Buffer,
			dialTimeout:       c.DialTimeout,
			connectionTimeout: c.ConnectionTimeout,
			runner:            c.Runner,
			cancel:            c.Cancel,
			noLocalAccess:     c.NoLocalAccess,
			rw:                rw,
			relay:             nil,
		},
		mapping: c.Mapping,
	}
}

func (c *tcpMapping) Bootup() (fsm.State, error) {
	_, rErr := io.ReadFull(c.rw, c.buf[:1])

	if rErr != nil {
		c.rw.Done()

		return nil, rErr
	}

	c.rw.Done()

	mapped, mappedErr := c.mapping.Get(common.MapID(c.buf[0]))

	if mappedErr != nil {
		c.rw.Write([]byte{TCPRespondMappingNotFound})

		return nil, mappedErr
	}

	if mapped.Protocol != network.TCP {
		c.rw.Write([]byte{TCPRespondMappingNotFound})

		return nil, ErrTCPMappingNotFound
	}

	c.relay = relay.New(c.logger, c.runner, c.rw, c.buf, tcpRelay{
		noLocalAccess:     c.noLocalAccess,
		dialTimeout:       c.dialTimeout,
		connectionTimeout: c.connectionTimeout,
		dial: tcpdial.New(
			mapped.Host, mapped.Port, c.dialTimeout, tcpconn.Wrap).Dialer(),
	}, make([]byte, 4096))

	bootErr := c.relay.Bootup(c.cancel)

	if bootErr != nil {
		return nil, bootErr
	}

	return c.tick, nil
}
