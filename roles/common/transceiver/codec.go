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

package transceiver

import "io"

// CodecBuilder creates a new Codec
type CodecBuilder func(io.ReadWriter) (io.ReadWriter, error)

// Codec is the registeration information of a CodecBuilder
type Codec struct {
	Name   string
	Usage  string
	Build  func(configuration []byte) CodecBuilder
	Verify func(configuration []byte) error
}

// CodecError is the error of codec
type CodecError interface {
	error
	Type() CodecError
}

type codecError struct {
	err string
}

func (c codecError) Error() string {
	return "Transceiver Codec Error: " + c.err
}

func (c codecError) Type() CodecError {
	return c
}

type codecErrorWrapper struct {
	err error
}

func (c codecErrorWrapper) Error() string {
	return "Transceiver Codec Error: " + c.err.Error()
}

func (c codecErrorWrapper) Type() CodecError {
	return c
}

// NewCodecError creates a new codec error
func NewCodecError(msg string) CodecError {
	return codecError{err: msg}
}

// WrapCodecError wraps an error as a codec error
func WrapCodecError(err error) CodecError {
	return codecErrorWrapper{err: err}
}
