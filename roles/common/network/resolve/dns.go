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

package resolve

import (
	"context"
	"errors"
	"net"
	"time"
)

// Errors
var (
	ErrDNSReverseUnsupported = errors.New(
		"Reverse is unsupported")

	ErrDNSResolvedNoResult = errors.New(
		"DNS resolves no result")
)

type dns struct {
	resolver net.Resolver
	timeout  time.Duration
}

// DNS returns a DNS Resolver
func DNS(timeout time.Duration) Resolver {
	return dns{
		resolver: net.Resolver{
			PreferGo:     true,
			StrictErrors: false,
			Dial:         nil,
		},
		timeout: timeout,
	}
}

func (d dns) Resolve(domain string) ([]net.IP, error) {
	ctx, cancel := context.WithDeadline(
		context.Background(), time.Now().Add(d.timeout))

	defer cancel()

	resolved, resolveErr := d.resolver.LookupIPAddr(ctx, domain)

	if resolveErr != nil {
		return nil, resolveErr
	}

	if len(resolved) <= 0 {
		return nil, ErrDNSResolvedNoResult
	}

	ips := make([]net.IP, len(resolved))

	for i := range resolved {
		ips[i] = resolved[i].IP
	}

	return ips, resolveErr
}

func (d dns) Reverse(ip net.IP) (string, error) {
	return "", nil
}
