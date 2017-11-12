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

package common

import "github.com/reinit/coward/roles/common/transceiver"

// SharedBuffer is a large trunk of memory that made for the Read operation
// of Transceiver Client
type SharedBuffer struct {
	Buffer []byte
	Size   uint32
}

// SharedBuffers is a group of SharedBuffer
type SharedBuffers struct {
	Buf []*SharedBuffer
}

// For selects the SharedBuffer for a Client
func (s SharedBuffers) For(id transceiver.ClientID) *SharedBuffer {
	return s.Buf[id]
}

// Select the buffer according to Transceiver Connection ID
func (s SharedBuffer) Select(id transceiver.ConnectionID) []byte {
	start := uint32(id) * s.Size

	return s.Buffer[start : start+s.Size]
}
