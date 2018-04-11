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

package codec

import (
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/codec/plain"
	"github.com/reinit/coward/roles/common/transceiver"
)

var (
	plainPrefixerOptions = []string{
		"Request-Prefix", "Respond-Prefix",
	}
)

const (
	plainPrefixerUsageErr = "You must define an option " +
		"before configuring it. Available options: %s.\r\n\r\nExample:" +
		"\r\n\r\nRequest-Prefix: 436C69656E74\r\nRespond-Prefix: 536572766572" +
		"\r\n\r\nNotice:\r\n * One single line can only contain " +
		"one option\r\n * The value of \"Request-Prefix\" and " +
		"\"Respond-Prefix\" option must be swapped at the opponent " +
		"side accordingly"
)

// Plain return a Plain Transceiver Codec
func Plain() transceiver.Codec {
	return transceiver.Codec{
		Name:   "plain",
		Usage:  "No option required",
		Build:  plainBuilder,
		Verify: plainVerifier,
	}
}

func plainVerifier(configuration []string) error {
	return nil
}

func plainBuilder(configuration []string) transceiver.CodecBuilder {
	return func() (rw.Codec, error) {
		return plain.New()
	}
}
