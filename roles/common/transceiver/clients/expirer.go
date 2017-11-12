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

package clients

import "github.com/reinit/coward/roles/common/transceiver"

type expirer struct {
	dests   []transceiver.Destination
	nextIdx int
	maxSize int
}

func (e *expirer) Bump(idx int) (transceiver.Destination, int) {
	curIdx := e.nextIdx
	out := e.dests[e.nextIdx]

	e.dests[e.nextIdx] = e.dests[idx]
	e.dests[idx] = ""

	if e.nextIdx+1 >= e.maxSize {
		e.nextIdx = 0
	} else {
		e.nextIdx++
	}

	return out, curIdx
}

func (e *expirer) Add(
	d transceiver.Destination) (transceiver.Destination, int) {
	curIdx := e.nextIdx
	out := e.dests[e.nextIdx]

	e.dests[e.nextIdx] = d

	if e.nextIdx+1 >= e.maxSize {
		e.nextIdx = 0
	} else {
		e.nextIdx++
	}

	return out, curIdx
}
