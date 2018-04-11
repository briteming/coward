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

package udp

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

	id                   network.ConnectionID
	readTimeout          time.Duration
	readTimeoutSet       bool
	readTimeoutDisabled  bool
	writeTimeout         time.Duration
	writeTimeoutSet      bool
	writeTimeoutDisabled bool
	close                chan struct{}
	closed               bool
	closeLock            sync.Mutex
}

// Wrap wraps a net.Conn to a network.Connection
func Wrap(conn net.Conn) network.Connection {
	return &connection{
		Conn: conn,
		id: network.ConnectionID(
			conn.LocalAddr().String() + "-" + conn.RemoteAddr().String()),
		readTimeout:          zeroTimeDuration,
		readTimeoutSet:       false,
		readTimeoutDisabled:  false,
		writeTimeout:         zeroTimeDuration,
		writeTimeoutSet:      false,
		writeTimeoutDisabled: false,
		close:                make(chan struct{}, 1),
		closed:               false,
		closeLock:            sync.Mutex{},
	}
}

func (c *connection) filterErr(err error) error {
	switch err {
	case io.EOF:
		c.drop()
	}

	return err
}

func (c *connection) drop() error {
	c.closeLock.Lock()
	defer c.closeLock.Unlock()

	if c.closed {
		return io.EOF
	}

	close(c.close)
	c.closed = true

	return c.Conn.Close()
}

func (c *connection) ID() network.ConnectionID {
	return c.id
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
	var totalReadLen int

	timeoutRetry := false
	resetTimeout := true
	toTryAgain := true

	for {
		if resetTimeout &&
			!c.readTimeoutDisabled && c.readTimeout != zeroTimeDuration {
			timeoutRetry = true

			if !c.readTimeoutSet {
				c.readTimeoutSet = true

				c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))

				toTryAgain = false
			}
		}

		rLen, rErr := c.Conn.Read(b[totalReadLen:])

		totalReadLen += rLen

		c.readTimeoutDisabled = false

		if rErr == nil {
			return totalReadLen, nil
		}

		if !timeoutRetry {
			return totalReadLen, c.filterErr(rErr)
		}

		resetTimeout = false
		timeoutRetry = false

		if err, ok := rErr.(net.Error); !ok || !err.Timeout() || !toTryAgain {
			return totalReadLen, c.filterErr(rErr)
		}

		c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))

		toTryAgain = false
	}
}

func (c *connection) Write(b []byte) (int, error) {
	var totalWriteLen int

	timeoutRetry := false
	resetTimeout := true
	toTryAgain := true

	for {
		if resetTimeout &&
			!c.writeTimeoutDisabled && c.writeTimeout != zeroTimeDuration {
			timeoutRetry = true

			if !c.writeTimeoutSet {
				c.writeTimeoutSet = true

				c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))

				toTryAgain = false
			}
		}

		wLen, wErr := c.Conn.Write(b[totalWriteLen:])

		totalWriteLen += wLen

		c.writeTimeoutDisabled = false

		if wErr == nil {
			return totalWriteLen, nil
		}

		if !timeoutRetry {
			return totalWriteLen, c.filterErr(wErr)
		}

		resetTimeout = false
		timeoutRetry = false

		if err, ok := wErr.(net.Error); !ok || !err.Timeout() || !toTryAgain {
			return totalWriteLen, c.filterErr(wErr)
		}

		c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))

		toTryAgain = false
	}
}

func (c *connection) Close() error {
	return c.drop()
}

func (c *connection) Closed() <-chan struct{} {
	return c.close
}
