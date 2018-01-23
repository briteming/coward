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

package prefixer

import (
	"bytes"
	"io"

	"github.com/reinit/coward/roles/common/transceiver"
)

// Errors
var (
	ErrUnexpectedRequestPrefix = transceiver.NewCodecError(
		"Unexpected Request Prefix")

	ErrFailedToReadRequestPrefix = transceiver.NewCodecError(
		"Failed to read Request Prefix")

	ErrFailedToWriteRespondPrefix = transceiver.NewCodecError(
		"Failed to read Respond Prefix")
)

// Consts
const (
	prefixBufSize = 256
)

// prefixer read or write a prefix before handle actual data
type prefixer struct {
	io.ReadWriter

	requestPrefix            []byte
	expectedRequestPrefix    []byte
	expectedRequestPrefixLen int
	respondWrite             bool
	requestPrefixReaded      bool
}

// New create a new prefixer read writer
func New(
	conn io.ReadWriter,
	requestPrefix []byte,
	expectedRequestPrefix []byte,
) io.ReadWriter {
	return &prefixer{
		ReadWriter:               conn,
		requestPrefix:            requestPrefix,
		expectedRequestPrefix:    expectedRequestPrefix,
		expectedRequestPrefixLen: len(expectedRequestPrefix),
		respondWrite:             false,
		requestPrefixReaded:      false,
	}
}

// Read reads the prefix before reading actual data
func (p *prefixer) Read(b []byte) (int, error) {
	if !p.requestPrefixReaded {
		p.requestPrefixReaded = true

		prefixBuf := [prefixBufSize]byte{}
		prefixBufReadLen := prefixBufSize
		prefixReaded := 0

		if prefixBufReadLen > p.expectedRequestPrefixLen {
			prefixBufReadLen = p.expectedRequestPrefixLen
		}

		for {
			rLen, rErr := p.ReadWriter.Read(prefixBuf[:prefixBufReadLen])

			if rErr != nil {
				return 0, ErrFailedToReadRequestPrefix
			}

			if !bytes.Equal(
				prefixBuf[:rLen],
				p.expectedRequestPrefix[prefixReaded:prefixReaded+rLen],
			) {
				return 0, ErrUnexpectedRequestPrefix
			}

			totalReadedLen := prefixReaded + rLen

			if totalReadedLen >= p.expectedRequestPrefixLen {
				break
			}

			prefixReaded = totalReadedLen
			prefixBufReadLen = p.expectedRequestPrefixLen - prefixReaded

			if prefixBufReadLen > prefixBufSize {
				prefixBufReadLen = prefixBufSize
			}
		}
	}

	return p.ReadWriter.Read(b)
}

// Write writes respond prefix before writing actual data
func (p *prefixer) Write(b []byte) (int, error) {
	if !p.respondWrite {
		p.respondWrite = true

		_, wErr := p.ReadWriter.Write(p.requestPrefix)

		if wErr != nil {
			return 0, ErrFailedToWriteRespondPrefix
		}
	}

	return p.ReadWriter.Write(b)
}
