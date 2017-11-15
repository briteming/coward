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

package mapper

import (
	"net"
	"time"

	"github.com/reinit/coward/roles/common/network"
	proxycomm "github.com/reinit/coward/roles/proxy/common"
)

// Mapped Item
type Mapped struct {
	ID        proxycomm.MapID
	Protocol  network.Protocol
	Interface net.IP
	Port      uint16
	Capacity  uint32
}

// Mappeds a group of Mapped
type Mappeds []Mapped

// TotalCapacity get total capacity of all Mapped
func (m Mappeds) TotalCapacity() uint32 {
	totalCap := uint32(0)

	for k := range m {
		totalCap += m[k].Capacity
	}

	return totalCap
}

// Config Configuration
type Config struct {
	TransceiverMaxConnections       uint32
	TransceiverConnectionPersistent bool
	TransceiverRequestRetries       uint8
	TransceiverIdleTimeout          time.Duration
	TransceiverInitialTimeout       time.Duration
	TransceiverChannels             uint8
	Mapping                         Mappeds
}
