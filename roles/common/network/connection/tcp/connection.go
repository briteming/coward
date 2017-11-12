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

package tcp

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/reinit/coward/roles/common/network"
)

const (
	zeroTimeDuration = time.Duration(0)
)

type connection struct {
	net.Conn

	readTimeout          time.Duration
	readTimeoutSet       bool
	readTimeoutDisabled  bool
	writeTimeout         time.Duration
	writeTimeoutSet      bool
	writeTimeoutDisabled bool
	close                chan struct{}
	closeLock            sync.Mutex
}

// Wrap wraps a net.Conn to a network.Connection
func Wrap(conn net.Conn) network.Connection {
	return &connection{
		Conn:                 conn,
		readTimeout:          zeroTimeDuration,
		readTimeoutSet:       false,
		readTimeoutDisabled:  false,
		writeTimeout:         zeroTimeDuration,
		writeTimeoutSet:      false,
		writeTimeoutDisabled: false,
		close:                make(chan struct{}, 1),
		closeLock:            sync.Mutex{},
	}
}

func (c *connection) filterErr(err error) error {
	switch err {
	case io.EOF:
		c.setDropped()
	}

	return err
}

func (c *connection) setDropped() {
	c.closeLock.Lock()
	defer c.closeLock.Unlock()

	select {
	case <-c.close:
		return

	default:
		close(c.close)
	}
}

func (c *connection) SetTimeout(t time.Duration) {
	c.SetReadTimeout(t)
	c.SetWriteTimeout(t)
}

func (c *connection) SetReadTimeout(t time.Duration) {
	c.readTimeout = t
}

func (c *connection) SetWriteTimeout(t time.Duration) {
	c.writeTimeout = t
}

func (c *connection) SetReadDeadline(t time.Time) error {
	c.readTimeoutDisabled = true

	return c.filterErr(c.Conn.SetReadDeadline(t))
}

func (c *connection) SetWriteDeadline(t time.Time) error {
	c.writeTimeoutDisabled = true

	return c.filterErr(c.Conn.SetWriteDeadline(t))
}

func (c *connection) SetDeadline(t time.Time) error {
	deadlineErr := c.SetReadDeadline(t)

	if deadlineErr != nil {
		return deadlineErr
	}

	deadlineErr = c.SetWriteDeadline(t)

	if deadlineErr != nil {
		return deadlineErr
	}

	return nil
}

func (c *connection) Read(b []byte) (int, error) {
	now := time.Now()
	timeoutRetry := false
	resetTimeout := true

	for {
		if resetTimeout &&
			!c.readTimeoutDisabled && c.readTimeout != zeroTimeDuration {
			timeoutRetry = true

			if !c.readTimeoutSet {
				c.readTimeoutSet = true

				c.Conn.SetReadDeadline(now.Add(c.readTimeout))
			}
		}

		rLen, rErr := c.Conn.Read(b)

		c.readTimeoutDisabled = false

		if rErr == nil {
			return rLen, nil
		}

		if !timeoutRetry {
			return rLen, c.filterErr(rErr)
		}

		resetTimeout = false
		timeoutRetry = false

		if err, ok := rErr.(net.Error); !ok || !err.Timeout() {
			return rLen, c.filterErr(rErr)
		}

		c.Conn.SetReadDeadline(now.Add(c.readTimeout))
	}
}

func (c *connection) Write(b []byte) (int, error) {
	now := time.Now()
	timeoutRetry := false
	resetTimeout := true

	for {
		if resetTimeout &&
			!c.writeTimeoutDisabled && c.writeTimeout != zeroTimeDuration {
			timeoutRetry = true

			if !c.writeTimeoutSet {
				c.writeTimeoutSet = true

				c.Conn.SetWriteDeadline(now.Add(c.writeTimeout))
			}
		}

		wLen, wErr := c.Conn.Write(b)

		c.writeTimeoutDisabled = false

		if wErr == nil {
			return wLen, nil
		}

		if !timeoutRetry {
			return wLen, c.filterErr(wErr)
		}

		resetTimeout = false
		timeoutRetry = false

		if err, ok := wErr.(net.Error); !ok || !err.Timeout() {
			return wLen, c.filterErr(wErr)
		}

		c.Conn.SetWriteDeadline(now.Add(c.writeTimeout))
	}
}

func (c *connection) Close() error {
	c.setDropped()

	return c.Conn.Close()
}

func (c *connection) Closed() <-chan struct{} {
	return c.close
}