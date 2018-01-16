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

package rw

import "io"

type byteSlice struct {
	buf    []byte
	remain int
}

// ByteSliceReader returns a reader of the input []byte
func ByteSliceReader(b []byte) io.Reader {
	return &byteSlice{
		buf:    b,
		remain: len(b),
	}
}

func (b *byteSlice) Read(bb []byte) (int, error) {
	if b.remain <= 0 {
		return 0, io.EOF
	}

	sizeToCopy := len(bb)
	currentStart := len(b.buf) - b.remain

	if sizeToCopy > b.remain {
		sizeToCopy = b.remain
	}

	copied := copy(bb, b.buf[currentStart:currentStart+sizeToCopy])

	b.remain -= copied

	return copied, nil
}
