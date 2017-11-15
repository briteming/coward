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

import (
	"errors"
	"net"
	"sync"

	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/roles/common/transceiver"
)

// Errors
var (
	ErrAlreadyBootedUp = errors.New(
		"Already booted up")

	ErrAlreadyClosed = errors.New(
		"Already closed")
)

type clients struct {
	clients      []transceiver.Client
	requesters   requesters
	destinations destinations
	bootLock     sync.RWMutex
	booted       bool
}

// New creates a new Transceiver Balancer
func New(clis []transceiver.Client, maxDestinations int) transceiver.Balancer {
	return &clients{
		clients: clis,
		requesters: requesters{
			req: make([]*requester, len(clis)),
		},
		destinations: destinations{
			dest: make(
				map[transceiver.Destination]*destination, maxDestinations),
			expire: expirer{
				dests:   make([]transceiver.Destination, maxDestinations),
				nextIdx: 0,
				maxSize: maxDestinations,
			},
		},
		bootLock: sync.RWMutex{},
		booted:   false,
	}
}

func (c *clients) Serve() (transceiver.Balanced, error) {
	c.bootLock.Lock()
	defer c.bootLock.Unlock()

	if c.booted {
		return nil, ErrAlreadyBootedUp
	}

	for cIdx := range c.clients {
		req, reqServErr := c.clients[cIdx].Serve()

		if reqServErr != nil {
			// Close all enabled clients
			for rCloseIdx := range c.requesters.req {
				if c.requesters.req[rCloseIdx].requester == nil {
					continue
				}

				c.requesters.req[rCloseIdx].requester.Close()
			}

			return nil, reqServErr
		}

		c.requesters.req[req.ID()] = &requester{
			id:        req.ID(),
			requester: req,
			sink:      false,
			delay:     timer.Average(),
		}
	}

	c.booted = true

	return c, nil
}

func (c *clients) Close() error {
	c.bootLock.Lock()
	defer c.bootLock.Unlock()

	if !c.booted {
		return ErrAlreadyClosed
	}

	// Close all requesters
	var closeErr error

	for rCloseIdx := range c.requesters.req {
		if c.requesters.req[rCloseIdx].requester == nil {
			continue
		}

		cErr := c.requesters.req[rCloseIdx].requester.Close()

		if cErr == nil {
			continue
		}

		closeErr = cErr
	}

	// Mark shutdown
	c.booted = false

	return closeErr
}

func (c *clients) Size() int {
	return len(c.requesters.req)
}

func (c *clients) Clients(r func(transceiver.ClientID, transceiver.Requester)) {
	for reqID := range c.requesters.req {
		r(c.requesters.req[reqID].ID(), c.requesters.req[reqID].requester)
	}
}

func (c *clients) Request(
	reqer net.Addr,
	dest transceiver.Destination,
	req transceiver.BalancedRequestBuilder,
	cancel <-chan struct{},
) error {
	return c.destinations.Request(
		reqer, dest, req, cancel, &c.requesters, &c.bootLock)
}
