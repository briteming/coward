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

package connection

// Error respresents a Connection Error
type Error interface {
	error
	Type() Error
	Get() error
}

// failure implements Error
type failure struct {
	err string
}

// err implements Error
type err struct {
	err error
}

// NewError creates a new Error
func NewError(message string) Error {
	return &failure{
		err: "Transceiver Connection Error: " + message,
	}
}

// WrapError wraps a existing error to Error
func WrapError(e error) Error {
	return &err{
		err: e,
	}
}

// Type get current Error type
func (f *failure) Type() Error {
	return f
}

// Get gets current Error as error
func (f *failure) Get() error {
	return f
}

// Error returns error message
func (f *failure) Error() string {
	return f.err
}

// Type get current Error type
func (f *err) Type() Error {
	return f
}

// Get gets current Error as error
func (f *err) Error() string {
	return "Transceiver Connection Error: " + f.err.Error()
}

// Error returns error message
func (f *err) Get() error {
	return f.err
}
