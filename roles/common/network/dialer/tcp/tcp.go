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

package tcp

import (
	"net"
	"strconv"
	"time"

	"github.com/reinit/coward/roles/common/network"
)

type dialer struct {
	host        string
	port        uint16
	timeout     time.Duration
	connWrapper network.ConnectionWrapper
}

type dial struct {
	resolved    net.IP
	useResolved bool
	host        string
	port        uint16
	timeout     time.Duration
	connWrapper network.ConnectionWrapper
}

// New returns a new TCP Dialer
func New(
	host string,
	port uint16,
	timeout time.Duration,
	connWrapper network.ConnectionWrapper,
) network.Dialer {
	return dialer{
		host:        host,
		port:        port,
		timeout:     timeout,
		connWrapper: connWrapper,
	}
}

func (d dialer) Dialer() network.Dial {
	return &dial{
		resolved:    nil,
		useResolved: false,
		host:        d.host,
		port:        d.port,
		timeout:     d.timeout,
		connWrapper: d.connWrapper,
	}
}

func (d *dial) resolvedAddress() string {
	var address string

	if d.resolved != nil && d.useResolved {
		address = net.JoinHostPort(
			d.resolved.String(), strconv.FormatUint(uint64(d.port), 10))
	} else {
		address = net.JoinHostPort(
			d.host, strconv.FormatUint(uint64(d.port), 10))
	}

	return address
}

func (d *dial) Dial() (network.Connection, error) {
	dialed, dialErr := net.DialTimeout("tcp", d.resolvedAddress(), d.timeout)

	if dialErr != nil {
		d.useResolved = !d.useResolved

		return nil, dialErr
	}

	d.useResolved = true
	d.resolved = dialed.RemoteAddr().(*net.TCPAddr).IP

	return d.connWrapper(dialed), nil
}

func (d *dial) String() string {
	return d.resolvedAddress()
}
