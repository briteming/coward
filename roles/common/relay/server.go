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

package relay

import (
	"errors"
	"fmt"
	"io"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
)

// Errors
var (
	ErrUnexpectedCloseSignal = errors.New(
		"Unexpected Relay close signal")
)

// Server is server-side conn wrapper for Relay Client initialization
type Server interface {
	rw.ReadWriteDepleteDoner

	Goodbye() error
}

// server implements Server
type server struct {
	rw.ReadWriteDepleteDoner

	log logger.Logger
}

func (c server) waitForRespond(checker func(s Signal) bool) (Signal, error) {
	defer c.ReadWriteDepleteDoner.Done()

	command := [1]byte{}

	for {
		_, crErr := io.ReadFull(c.ReadWriteDepleteDoner, command[:])

		if crErr != nil {
			return 0, crErr
		}

		signalRead := Signal(command[0])

		if checker(signalRead) {
			return signalRead, nil
		}

		c.ReadWriteDepleteDoner.Done()
	}
}

// Goodbye properly asks remote relay to shutdown
func (c server) Goodbye() error {
	c.log.Debugf("Sending Goodbye")

	_, wErr := rw.WriteFull(c.ReadWriteDepleteDoner, []byte{
		byte(SignalCompleted)})

	if wErr != nil {
		return wErr
	}

	signalPassed := 0

	respondedSignal, respErr := c.waitForRespond(func(s Signal) bool {
		signalPassed++

		if s != SignalClose && s != SignalClosed {
			c.log.Debugf("Ignoring remote signal \"%s\"", s)

			return false
		}

		return true
	})

	if respErr != nil {
		return respErr
	}

	c.log.Debugf("Reached expected remote confirm signal \"%s\" after reading "+
		"%d segments", respondedSignal, signalPassed)

	switch respondedSignal {
	case SignalClosed:
		return nil

	case SignalClose:
		_, wErr = rw.WriteFull(c.ReadWriteDepleteDoner, []byte{
			byte(SignalClosed)})

		return wErr
	}

	return ErrUnexpectedCloseSignal
}

// Write perform some operations to prepare the Relay data, and then, send
// them.
func (c server) Write(b []byte) (int, error) {
	if len(b) <= 0 {
		return 0, nil
	}

	switch Signal(b[0]) {
	case SignalData:
		fallthrough
	case SignalError:
		fallthrough
	case SignalCompleted:
		fallthrough
	case SignalClose:
		panic(fmt.Sprintf("\"%d\" as in data %d are reserved Relay Signal "+
			"code. Use of it is not allowed.", b[0], b))
	}

	return c.ReadWriteDepleteDoner.Write(b)
}
