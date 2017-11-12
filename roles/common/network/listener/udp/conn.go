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
	"io"
	"net"
	"time"
)

// Errors
var (
	ErrReadTimeout = errors.New(
		"UDP Read has timed out")
)

var (
	emptyTime = time.Time{}
)

type conn struct {
	conn                *net.UDPConn
	addr                *net.UDPAddr
	clients             *clients
	deadlineTicker      <-chan time.Time
	ipPort              ipport
	currentReader       *rbuffer
	readerDeliver       chan *rbuffer
	readDeadline        time.Time
	readDeadlineEnabled bool
	closed              bool
}

func (c *conn) RemoteAddr() net.Addr {
	return c.addr
}

func (c *conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *conn) SetDeadline(t time.Time) error {
	c.conn.SetWriteDeadline(t)
	c.SetReadDeadline(t)

	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t

	if t == emptyTime {
		c.readDeadlineEnabled = false
	} else {
		c.readDeadlineEnabled = true
	}

	// Don't set timeout of the main listener conn

	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *conn) getReader() (io.Reader, error) {
	if c.currentReader != nil {
		return c.currentReader, nil
	}

	for {
		checkTicker := c.deadlineTicker

		if !c.readDeadlineEnabled {
			checkTicker = nil
		}

		select {
		case <-checkTicker:
			if time.Now().After(c.readDeadline) {
				return nil, ErrReadTimeout
			}

		case r, rOK := <-c.readerDeliver:
			if !rOK {
				return nil, io.EOF
			}

			return r, nil
		}
	}
}

func (c *conn) Read(b []byte) (int, error) {
	if c.closed {
		return 0, io.EOF
	}

	for {
		reader, readerErr := c.getReader()

		if readerErr == io.EOF {
			c.closed = true

			return 0, io.EOF
		}

		if readerErr != nil {
			return 0, readerErr
		}

		rLen, rErr := reader.Read(b)

		if rErr == errRBufferNoMoreData {
			c.currentReader = nil

			continue
		}

		return rLen, rErr
	}
}

func (c *conn) Write(b []byte) (int, error) {
	return c.conn.WriteToUDP(b, c.addr)
}

func (c *conn) Kick() error {
	select {
	case n := <-c.readerDeliver:
		close(c.readerDeliver)

		n.Clear()

	default:
		close(c.readerDeliver)

		if c.currentReader != nil {
			c.currentReader.Clear()
		}

		c.currentReader = nil
	}

	return nil
}

func (c *conn) Close() error {
	return c.clients.Delete(c.ipPort, func() error {
		kickErr := c.Kick()

		if kickErr != nil {
			return kickErr
		}

		return nil
	})
}
