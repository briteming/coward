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

import (
	"time"

	"github.com/reinit/coward/common/timer"
)

// Receiver represents a Projection Receiver
type Receiver interface {
	ID() ID
	Receive() chan Accessor
	Expand()
	Shrink()
	Delay(delay time.Duration)
	Remove()
}

// receiver implements Receiver
type receiver struct {
	plot         *receivers
	previous     *receiver
	next         *receiver
	id           ID
	pingTimer    timer.Timer
	accessorChan chan Accessor
	delay        time.Duration
	capcity      uint32
	lifted       bool
	deleted      bool
}

// linkAfter links current node after given node
func (p *receiver) linkAfter(to *receiver) {
	p.previous = to
	p.next = to.next

	to.next = p

	if p.next != nil {
		p.next.previous = p
	} else {
		p.plot.Tail = p
	}
}

// linkBefore links current node before given node
func (p *receiver) linkBefore(to *receiver) {
	p.previous = to.previous
	p.next = to

	to.previous = p

	if p.previous != nil {
		p.previous.next = p
	} else {
		p.plot.Head = p
	}
}

// unlink unchain current node from the node chain
func (p *receiver) unlink() {
	if p.previous != nil {
		p.previous.next = p.next
	}

	if p.next != nil {
		p.next.previous = p.previous
	}

	if p == p.plot.Head {
		p.plot.Head = p.next
	}

	if p == p.plot.Tail {
		p.plot.Tail = p.previous
	}
}

// remove removes the receiver from current chain and reduce the capcity
func (p *receiver) remove() {
	p.unlink()

	p.previous = nil
	p.next = nil

	<-p.plot.Capcity
}

// insert inserts the node to the node chain
func (p *receiver) insert() {
	p.linkAfter(p.plot.Tail)

	p.plot.Capcity <- struct{}{}
}

// insertByDelay inserts the node to the node chain on delay order
func (p *receiver) insertByDelay() {
	current := p.plot.Head

	for {
		if current.next == nil || current.next.delay > p.delay {
			break
		}

		current = current.next
	}

	p.linkBefore(current)

	p.plot.Capcity <- struct{}{}
}

// ID returns the ID of current receiver
func (p *receiver) ID() ID {
	return p.id
}

// Receive returns current Accessor receiving chan
func (p *receiver) Receive() chan Accessor {
	return p.accessorChan
}

// Expand declares increase capcity for the current receiver
func (p *receiver) Expand() {
	p.plot.Capacitor.L.Lock()
	defer p.plot.Capacitor.L.Unlock()

	p.capcity++
}

// Shrink declares decrease capcity for the current receiver
func (p *receiver) Shrink() {
	p.plot.Capacitor.L.Lock()
	defer p.plot.Capacitor.L.Unlock()

	p.capcity--
}

// Delay sets the delay of current receiver
func (p *receiver) Delay(delay time.Duration) {
	p.plot.Capacitor.L.Lock()
	defer p.plot.Capacitor.L.Unlock()

	p.delay = delay

	// If p.flateLevel == 0, we may not attached with the chain, so
	// don't update our position on the chain
	if p.lifted || p.deleted {
		return
	}

	if p.previous != nil && p.previous.delay > p.delay { // Towards head
		current := p.previous

		for {
			if current.previous == nil || current.previous.delay < p.delay {
				break
			}

			current = current.previous
		}

		p.unlink()
		p.linkBefore(current)
	} else if p.next != nil && p.next.delay < p.delay { // Towards tail
		current := p.next

		for {
			if current.next == nil || current.next.delay > p.delay {
				break
			}

			current = current.next
		}

		p.unlink()
		p.linkAfter(current)
	}
}

// ShrinkAndRemove usually shrinks the last receiver and remove it from chain
func (p *receiver) Remove() {
	p.plot.Capacitor.L.Lock()
	defer p.plot.Capacitor.L.Unlock()

	if p.deleted {
		return
	}

	p.deleted = true

	// Initial capcity is 1, must wait until we reached that capcity to
	// ensure safe deletation
	for p.capcity <= 0 || p.lifted {
		p.plot.Capacitor.Wait()
	}

	p.remove()
}
