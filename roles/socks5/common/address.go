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

package common

import (
	"errors"
	"io"
)

// Errors
var (
	ErrAddressInvalidSocks5AddressType = errors.New(
		"Invalid Socks5 address type")
)

// AType Address Type
type AType byte

// Consts
const (
	ATypeIPv4 AType = 0x01
	ATypeHost AType = 0x03
	ATypeIPv6 AType = 0x04
)

// Address is the Socks5 Address codec
type Address struct {
	AType   AType
	Address []byte
	Port    uint16
}

// ReadFrom read from reader
func (a *Address) ReadFrom(reader io.Reader) (int64, error) {
	buf := [2]byte{}

	rLen, rErr := io.ReadFull(reader, buf[:])

	if rErr != nil {
		return int64(rLen), rErr
	}

	switch AType(buf[0]) {
	case ATypeIPv4:
		aBuf := make([]byte, 6)

		arLen, arErr := io.ReadFull(reader, aBuf[1:])

		if arErr == nil {
			a.AType = ATypeIPv4
			a.Address = aBuf[0:4]
			a.Address[0] = buf[1]

			a.Port |= uint16(aBuf[4])
			a.Port <<= 8
			a.Port |= uint16(aBuf[5])
		}

		return int64(arLen + rLen), arErr

	case ATypeIPv6:
		aBuf := make([]byte, 18)

		arLen, arErr := io.ReadFull(reader, aBuf[1:])

		if arErr == nil {
			a.AType = ATypeIPv6
			a.Address = aBuf[0:16]
			a.Address[0] = buf[1]

			a.Port |= uint16(aBuf[16])
			a.Port <<= 8
			a.Port |= uint16(aBuf[17])
		}

		return int64(arLen + rLen), arErr

	case ATypeHost:
		aBuf := make([]byte, buf[1]+2)

		arLen, arErr := io.ReadFull(reader, aBuf)

		if arErr == nil {
			a.AType = ATypeHost
			a.Address = aBuf[:buf[1]]

			a.Port |= uint16(aBuf[buf[1]])
			a.Port <<= 8
			a.Port |= uint16(aBuf[buf[1]+1])
		}

		return int64(arLen + rLen), arErr

	default:
		return int64(rLen), ErrAddressInvalidSocks5AddressType
	}
}
