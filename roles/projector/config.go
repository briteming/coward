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

package projector

import (
	"net"
	"time"

	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/projector/projection"
)

// Server contains server data
type Server struct {
	ID             projection.ID
	Interface      net.IP
	Port           uint16
	Timeout        time.Duration
	RequestTimeout time.Duration
	Protocol       network.Protocol
	Capacity       uint32
}

// Config Configuration
type Config struct {
	Servers              []Server
	Capacity             uint32
	InitialTimeout       time.Duration
	IdleTimeout          time.Duration
	ConnectionChannels   uint8
	ChannelDispatchDelay time.Duration
}

// GetAllServerIDs returns the ID of all servers in an array
func (c Config) GetAllServerIDs() []projection.ID {
	ids := make([]projection.ID, len(c.Servers))

	for iIndex := range c.Servers {
		ids[iIndex] = c.Servers[iIndex].ID
	}

	return ids
}

// GetTotalServerCapacity returns total server capacity
func (c Config) GetTotalServerCapacity() uint32 {
	caps := uint32(0)

	for k := range c.Servers {
		caps += c.Servers[k].Capacity
	}

	return caps
}
