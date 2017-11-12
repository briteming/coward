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

package network

import "errors"

// Protocol is the network procotol
type Protocol uint8

// Errors
var (
	ErrorProtocolUnknown = errors.New(
		"Unknown Protocol")
)

// Consts
const (
	UnspecifiedProto Protocol = 0x00
	TCP              Protocol = 0x01
	UDP              Protocol = 0x02
)

// FromString select Protocol from a string
func (p *Protocol) FromString(n string) error {
	switch n {
	case "tcp":
		*p = TCP

	case "udp":
		*p = UDP

	default:
		return ErrorProtocolUnknown
	}

	return nil
}

// String return the String of current Protocol
func (p Protocol) String() string {
	switch p {
	case TCP:
		return "TCP"

	case UDP:
		return "UDP"

	default:
		return ""
	}
}
