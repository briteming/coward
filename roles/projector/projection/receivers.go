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

package projection

import "sync"

// receivers contains receiver chain data
type receivers struct {
	Head      *receiver
	Tail      *receiver
	Capcity   chan struct{}
	Capacitor sync.Cond
}

// insertOnDelayOrder inserts the given node to the chain according to
// it's delay
func (p *receivers) InsertDelayOrder(n *receiver) {
	if p.Head == nil || p.Tail == nil {
		p.Head = n
		p.Tail = n

		p.Capcity <- struct{}{}

		return
	}

	n.insertByDelay()
}

// Insert inserts given receiver to the tail of the receiver chain
func (p *receivers) Insert(n *receiver) {
	if p.Head == nil || p.Tail == nil {
		p.Head = n
		p.Tail = n

		p.Capcity <- struct{}{}

		return
	}

	n.insert()
}
