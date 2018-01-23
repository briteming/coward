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

import (
	"errors"
	"math"
	"sync"
	"time"
)

// Errors
var (
	ErrAlreadyServing = errors.New(
		"Already serving")

	ErrNotServing = errors.New(
		"Not serving")

	ErrUnoperatable = errors.New(
		"Ticker does not ready for operate")
)

// Ticker tick
type Ticker interface {
	Serve() (RequestCloser, error)
}

// Requester send request
type Requester interface {
	Request(when time.Time) (Waiter, error)
}

// RequestCloser send request and close Ticker
type RequestCloser interface {
	Requester

	Close() error
}

// request Request data
type request struct {
	When   time.Time
	Result chan Waiter
}

// ticker implements Ticker
type ticker struct {
	delay        time.Duration
	request      chan request
	closed       chan struct{}
	booted       bool
	bootLock     sync.Mutex
	shutdownWait sync.WaitGroup
}

// New creates a new Ticker
func New(delay time.Duration, reqBuffer uint32) Ticker {
	return &ticker{
		delay:        delay,
		request:      make(chan request, reqBuffer),
		closed:       nil,
		booted:       false,
		bootLock:     sync.Mutex{},
		shutdownWait: sync.WaitGroup{},
	}
}

// Serve startup Ticker
func (t *ticker) Serve() (RequestCloser, error) {
	t.bootLock.Lock()
	defer t.bootLock.Unlock()

	if t.booted {
		return nil, ErrAlreadyServing
	}

	t.closed = make(chan struct{})
	t.booted = true

	t.shutdownWait.Add(1)
	go t.ticking()

	return t, nil
}

// ticking handles request and ticks
func (t *ticker) ticking() {
	defer t.shutdownWait.Done()

	timeTicker := time.NewTicker(t.delay)
	defer timeTicker.Stop()

	var timeTickerChan <-chan time.Time

	wChain := waiters{
		Head:      nil,
		Tail:      nil,
		Canceller: make(chan *waiter),
	}
	nextTickTime := int64(math.MinInt64)

	for {
		select {
		case req := <-t.request:
			req.Result <- wChain.Insert(t.delay, req.When)

			nextTickTime = wChain.Head.When
			timeTickerChan = timeTicker.C

		case w := <-wChain.Canceller:
			w.delete()

		case <-t.closed:
			return

		case now := <-timeTickerChan:
			current := now.UnixNano()

			if current < nextTickTime {
				continue
			}

			var lastUsed *waiter

			wChain.Retrieve(func(ww *waiter) bool {
				if ww.When > current {
					return false
				}

				ww.Disabled = true

				close(ww.Signal)

				lastUsed = ww

				return true
			})

			if lastUsed != nil {
				wChain.UnlinkBy(lastUsed)
			}

			if wChain.Head != nil {
				nextTickTime = wChain.Head.When

				continue
			}

			nextTickTime = int64(math.MinInt64)
			timeTickerChan = nil
		}
	}
}

// Request send request
func (t *ticker) Request(when time.Time) (Waiter, error) {
	result := make(chan Waiter, 1)

	select {
	case t.request <- request{
		When:   when,
		Result: result,
	}:
		select {
		case wt := <-result:
			return wt, nil
		case <-t.closed:
			return nil, ErrUnoperatable
		}

	case <-t.closed:
		return nil, ErrUnoperatable
	}
}

// Close stop serving
func (t *ticker) Close() error {
	t.bootLock.Lock()
	defer t.bootLock.Unlock()

	if !t.booted {
		return ErrNotServing
	}

	close(t.closed)

	t.shutdownWait.Wait()

	t.booted = false

	return nil
}
