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

package aescfb

import (
	"crypto/rand"
	"io"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/transceiver"
)

// Errors
var (
	ErrPaddingTooLong = transceiver.NewCodecError(
		"Padding too long")
)

const (
	maxPaddingLength = 32
)

type padding struct {
	padBuf [maxPaddingLength]byte
}

func (p *padding) Passthrough(r io.Reader) error {
	_, rErr := io.ReadFull(r, p.padBuf[:1])

	if rErr != nil {
		return rErr
	}

	if p.padBuf[0] <= 0 {
		return nil
	}

	if p.padBuf[0] > maxPaddingLength {
		return ErrPaddingTooLong
	}

	_, rErr = io.ReadFull(r, p.padBuf[:p.padBuf[0]])

	if rErr != nil {
		return rErr
	}

	return nil
}

func (p *padding) Insert(w io.Writer) error {
	rand.Read(p.padBuf[:])

	p.padBuf[0] %= maxPaddingLength

	_, wErr := rw.WriteFull(w, p.padBuf[:1+p.padBuf[0]])

	return wErr
}
