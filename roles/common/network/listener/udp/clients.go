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

package udp

import (
	"errors"
	"sync"
)

// Errors
var (
	ErrClientsDeleteClientNotFound = errors.New(
		"Failed to delete the non-existed client")
)

type clients struct {
	clients map[ipport]*conn
	lock    sync.Mutex
	maxSize uint32
}

func (c *clients) Size() uint32 {
	c.lock.Lock()
	defer c.lock.Unlock()

	return uint32(len(c.clients))
}

func (c *clients) Fetch(ipPort ipport, run func(*conn), builder func() *conn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	client, found := c.clients[ipPort]

	if !found {
		client = builder()

		c.clients[ipPort] = client
	}

	run(client)
}

func (c *clients) Delete(ipPort ipport, callback func() error) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, found := c.clients[ipPort]

	if !found {
		return ErrClientsDeleteClientNotFound
	}

	delete(c.clients, ipPort)

	return callback()
}

func (c *clients) Clear(clear func(*conn)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	clientKeys := make([]ipport, len(c.clients))
	clientIndex := 0

	for k := range c.clients {
		clientKeys[clientIndex] = k

		clientIndex++
	}

	for v := range clientKeys {
		delete(c.clients, clientKeys[v])
	}
}
