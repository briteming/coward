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

package relay

import (
	"fmt"
	"io"

	"github.com/reinit/coward/common/rw"
)

// Server is server-side conn wrapper for Relay Client initialization
type Server interface {
	rw.ReadWriteDepleteDoner

	SendSignal(s Signal, expectingRespond Signal) error
}

// server implements Server
type server struct {
	rw.ReadWriteDepleteDoner
}

func (c server) waitForRespond(expecting Signal) error {
	command := [1]byte{}

	for {
		_, crErr := io.ReadFull(c.ReadWriteDepleteDoner, command[:])

		if crErr != nil {
			c.ReadWriteDepleteDoner.Done()

			return crErr
		}

		if Signal(command[0]) != expecting {
			c.ReadWriteDepleteDoner.Done()

			continue
		}

		c.ReadWriteDepleteDoner.Done()

		break
	}

	return nil
}

// SendSignalWaitRespond sends a Relay Signal, and wait until expected respond
// are received
func (c server) SendSignal(s Signal, expectingRespond Signal) error {
	_, wErr := rw.WriteFull(c.ReadWriteDepleteDoner, []byte{
		byte(s), byte(expectingRespond)})

	if wErr != nil {
		return wErr
	}

	return c.waitForRespond(expectingRespond)
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
