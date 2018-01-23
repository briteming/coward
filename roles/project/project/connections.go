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

package project

import (
	"time"

	"github.com/reinit/coward/roles/common/transceiver"
)

const (
	connectionPingTickDelay = 333 * time.Millisecond
)

type connectionPingTickPermit struct {
	Resume chan struct{}
	Next   time.Time
}

type connection struct {
	Buffer         []byte
	PingTickPermit chan connectionPingTickPermit
}

type connections struct {
	connections map[transceiver.ConnectionID]connection
}

func (c *connections) Get(cID transceiver.ConnectionID) connection {
	cc, ccFound := c.connections[cID]

	if ccFound {
		return connection{
			Buffer:         cc.Buffer,
			PingTickPermit: cc.PingTickPermit,
		}
	}

	cc = connection{
		Buffer:         make([]byte, 4096),
		PingTickPermit: make(chan connectionPingTickPermit, 1),
	}

	cc.PingTickPermit <- connectionPingTickPermit{
		Resume: make(chan struct{}, 1),
		Next:   time.Time{},
	}

	c.connections[cID] = cc

	return cc
}

func (c *connections) Clear() {
	deleteIdx := 0
	deleteKeys := make([]transceiver.ConnectionID, len(c.connections))

	for cKey := range c.connections {
		close(c.connections[cKey].PingTickPermit)

		deleteKeys[deleteIdx] = cKey
		deleteIdx++
	}

	for cKeyIdx := range deleteKeys {
		delete(c.connections, deleteKeys[cKeyIdx])
	}
}
