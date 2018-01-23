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

package ticker

import "time"

// Waiter waits or cancels ticker
type Waiter interface {
	Wait() Wait
	Close()
}

// Wait wait signal
type Wait chan struct{}

// waiters waiter data
type waiters struct {
	Head      *waiter
	Tail      *waiter
	Canceller chan *waiter
}

// UnlinkBy unlink inputted node and all nodes before it from the
// node chain
func (w *waiters) UnlinkBy(s *waiter) {
	if s.Next == nil {
		w.Head = nil
		w.Tail = nil
	} else {
		w.Head = s.Next

		s.Next.Previous = nil
		s.Next = nil
	}

	current := s

	for {
		current.Disabled = true
		current.Next = nil

		if current.Previous == nil {
			break
		}

		current = current.Previous
		current.Next.Previous = nil
	}
}

// Insert creates and adds waiter to the chan
func (w *waiters) Insert(truncate time.Duration, when time.Time) *waiter {
	whenTime := when.Add(truncate).Truncate(truncate).UnixNano()

	if w.Head == nil || w.Tail == nil {
		w.Head = &waiter{
			Plot:     w,
			Previous: nil,
			Next:     nil,
			Capacity: 1,
			Disabled: false,
			When:     whenTime,
			Signal:   make(Wait),
		}

		w.Tail = w.Head

		return w.Head
	}

	if w.Head.When > whenTime { // Earlier than the Head
		w.Head = &waiter{
			Plot:     w,
			Previous: nil,
			Next:     w.Head,
			Capacity: 1,
			Disabled: false,
			When:     whenTime,
			Signal:   make(Wait),
		}

		if w.Head.Next != nil {
			w.Head.Next.Previous = w.Head
		}

		return w.Head
	}

	if w.Tail.When < whenTime { // Later than the Tail
		w.Tail = &waiter{
			Plot:     w,
			Previous: w.Tail,
			Next:     nil,
			Capacity: 1,
			Disabled: false,
			When:     whenTime,
			Signal:   make(Wait),
		}

		if w.Tail.Previous != nil {
			w.Tail.Previous.Next = w.Tail
		}

		return w.Tail
	}

	whenHeadDiff := whenTime - w.Head.When
	whenEndDiffDiff := w.Tail.When - whenTime

	if whenHeadDiff <= whenEndDiffDiff {
		current := w.Head

		for {
			if current.When == whenTime {
				current.Capacity++

				return current
			}

			if current.When > whenTime {
				current.Previous = &waiter{
					Plot:     w,
					Previous: current.Previous,
					Next:     current,
					Capacity: 1,
					Disabled: false,
					When:     whenTime,
					Signal:   make(Wait),
				}

				current.Previous.Previous.Next = current.Previous

				return current.Previous
			}

			current = current.Next
		}
	}

	current := w.Tail

	for {
		if current.When == whenTime {
			current.Capacity++

			return current
		}

		if current.When < whenTime {
			current.Next = &waiter{
				Plot:     w,
				Previous: current,
				Next:     current.Next,
				Capacity: 1,
				Disabled: false,
				When:     whenTime,
				Signal:   make(Wait),
			}

			current.Next.Next.Previous = current.Next

			return current.Next
		}

		current = current.Previous
	}
}

// Retrieve retrieves waiter from chain head
func (w *waiters) Retrieve(read func(ww *waiter) bool) {
	current := w.Head

	for {
		if current == nil {
			break
		}

		if !read(current) {
			break
		}

		current = current.Next
	}
}

// waiter implements Waiter
type waiter struct {
	Plot     *waiters
	Previous *waiter
	Next     *waiter
	Capacity uint64
	Disabled bool
	When     int64
	Signal   chan struct{}
}

// unlink unchain current node from the node chain
func (p *waiter) unlink() {
	if p.Previous != nil {
		p.Previous.Next = p.Next
	}

	if p.Next != nil {
		p.Next.Previous = p.Previous
	}

	if p == p.Plot.Head {
		p.Plot.Head = p.Next
	}

	if p == p.Plot.Tail {
		p.Plot.Tail = p.Previous
	}
}

// delete clears up waiter capacity and cancels waiter when needed
func (p *waiter) delete() {
	if p.Capacity <= 0 {
		panic("Waiter Capacity must be greater than 0")
	}

	p.Capacity--

	if p.Capacity > 0 {
		return
	}

	p.unlink()
}

// Wait returns the Wait signal channel
func (p *waiter) Wait() Wait {
	return p.Signal
}

// Cancel cancel wait
func (p *waiter) Close() {
	p.Plot.Canceller <- p
}
