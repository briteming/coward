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
)

const (
	maxPaddingLength = 64
)

type padding struct {
	nextPaddingLength byte
}

func (p *padding) Passthrough(r io.Reader) error {
	if p.nextPaddingLength > 0 {
		padBuf := make([]byte, p.nextPaddingLength)

		rLen, rErr := io.ReadFull(r, padBuf)

		if rErr != nil {
			return rErr
		}

		if rLen > 0 {
			p.nextPaddingLength = padBuf[rLen-1]
		} else {
			p.nextPaddingLength = 0
		}

		return nil
	}

	passBuf := [1]byte{}
	initPaddingReaded := 0

	for {
		_, rErr := io.ReadFull(r, passBuf[:])

		if rErr != nil {
			return rErr
		}

		// If we've reached the maxPaddingLength, sliently and
		// intentionally assume we readed all padding. In this way
		// we can create some confusions for cracker
		if passBuf[0]&1 == 0 && initPaddingReaded < maxPaddingLength {
			continue
		}

		// If we have reached the end of padding, read next byte so
		// we can know the length of next padding
		_, rErr = io.ReadFull(r, passBuf[:])

		if rErr != nil {
			return rErr
		}

		p.nextPaddingLength = passBuf[0]

		return nil
	}
}

func (p *padding) Insert(w io.Writer, recommendedMaxLen byte) error {
	if p.nextPaddingLength > 0 {
		buf := make([]byte, p.nextPaddingLength)

		_, rErr := rand.Read(buf)

		if rErr != nil {
			return rErr
		}

		if recommendedMaxLen <= 0 {
			buf[p.nextPaddingLength-1] = 0
		} else if buf[p.nextPaddingLength-1] > recommendedMaxLen {
			buf[p.nextPaddingLength-1] %= recommendedMaxLen
		}

		p.nextPaddingLength = buf[p.nextPaddingLength-1]

		_, wErr := rw.WriteFull(w, buf)

		return wErr
	}

	// Build initial padding
	buf := [maxPaddingLength]byte{}

	_, rErr := rand.Read(buf[:])

	if rErr != nil {
		return rErr
	}

	paddingLen := (buf[0] % (maxPaddingLength - 2)) + 1

	for pIdx := range buf[:paddingLen-1] {
		buf[pIdx] &^= 1
	}

	buf[paddingLen-1] |= 1

	if recommendedMaxLen <= 0 {
		buf[paddingLen] = 0
	} else if buf[paddingLen] > recommendedMaxLen {
		buf[paddingLen] %= recommendedMaxLen
	}

	p.nextPaddingLength = buf[paddingLen]

	_, wErr := rw.WriteFull(w, buf[:paddingLen+1])

	return wErr
}
