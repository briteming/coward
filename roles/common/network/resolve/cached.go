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

package resolve

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// Errors
var (
	ErrCachedResolveNoResult = errors.New(
		"Resolve request returns no result")

	ErrCachedReverseNoResult = errors.New(
		"No host was found with specified IP")

	ErrCachedCacheTooMany = errors.New(
		"Too many cached results")
)

type cachedItem struct {
	Addresses []net.IP
	Expire    time.Time
}

type cached struct {
	cache          map[string]cachedItem
	reverseCache   map[IPMark]string
	cacheTTL       time.Duration
	maxSize        uint32
	resolver       net.Resolver
	resolveTimeout time.Duration
	lock           sync.RWMutex
}

// Cached returns a cached Resolver
func Cached(
	cacheTTL time.Duration,
	resolveTimeout time.Duration,
	maxSize uint32,
) Resolver {
	return &cached{
		cache:        make(map[string]cachedItem, maxSize),
		reverseCache: make(map[IPMark]string, maxSize),
		cacheTTL:     cacheTTL,
		maxSize:      maxSize,
		resolver: net.Resolver{
			PreferGo:     true,
			StrictErrors: false,
			Dial:         nil,
		},
		resolveTimeout: resolveTimeout,
		lock:           sync.RWMutex{},
	}
}

func (c *cached) Resolve(domain string) ([]net.IP, error) {
	c.lock.RLock()

	ip, found := c.cache[domain]

	if found {
		if !ip.Expire.After(time.Now()) {
			addresses := make([]net.IP, len(ip.Addresses))

			copy(addresses, ip.Addresses)

			c.lock.RUnlock()

			return addresses, nil
		}

		c.lock.RUnlock()
	} else if uint32(len(c.cache)) >= c.maxSize {
		c.lock.RUnlock()

		return nil, ErrCachedCacheTooMany
	}

	c.lock.RUnlock()

	ctx, cancel := context.WithDeadline(
		context.Background(), time.Now().Add(c.resolveTimeout))

	defer cancel()

	resolved, resolveErr := c.resolver.LookupIPAddr(ctx, domain)

	if resolveErr != nil {
		return nil, resolveErr
	}

	if len(resolved) <= 0 {
		return nil, ErrCachedResolveNoResult
	}

	c.lock.Lock()

	newResolve := cachedItem{
		Addresses: make([]net.IP, len(resolved)),
		Expire:    time.Now().Add(c.cacheTTL),
	}

	for rIdx := range resolved {
		newResolve.Addresses[rIdx] = resolved[rIdx].IP

		ipMark := IPMark{}
		ipMark.Import(resolved[rIdx].IP)

		c.reverseCache[ipMark] = domain
	}

	c.cache[domain] = newResolve

	c.lock.Unlock()

	return newResolve.Addresses, nil
}

func (c *cached) Reverse(ip net.IP) (string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ipMark := IPMark{}
	ipMark.Import(ip)

	domain, found := c.reverseCache[ipMark]

	if !found {
		return "", ErrCachedReverseNoResult
	}

	return domain, nil
}
