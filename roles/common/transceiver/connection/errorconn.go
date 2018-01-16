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

package connection

import (
	"time"

	"github.com/reinit/coward/roles/common/network"
)

// errorconn wraps a Connection, and convert it's error
// return to a connection.Error
type errorconn struct {
	network.Connection
}

func (e errorconn) Read(b []byte) (int, error) {
	l, err := e.Connection.Read(b)

	if err != nil {
		return l, WrapError(err)
	}

	return l, nil
}

func (e errorconn) Write(b []byte) (int, error) {
	l, err := e.Connection.Write(b)

	if err != nil {
		return l, WrapError(err)
	}

	return l, nil
}

func (e errorconn) Close() error {
	err := e.Connection.Close()

	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (e errorconn) SetDeadline(t time.Time) error {
	err := e.Connection.SetDeadline(t)

	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (e errorconn) SetReadDeadline(t time.Time) error {
	err := e.Connection.SetReadDeadline(t)

	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (e errorconn) SetWriteDeadline(t time.Time) error {
	err := e.Connection.SetWriteDeadline(t)

	if err != nil {
		return WrapError(err)
	}

	return nil
}
