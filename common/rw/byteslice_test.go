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

import (
	"io"
	"testing"
)

func TestByteSlice(t *testing.T) {
	example := []byte("AAAAAAAAAABBBBBBBBBBCCCCCCCCCC")
	r := ByteSliceReader(example)

	buf := [10]byte{}

	_, rErr := io.ReadFull(r, buf[:])

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if string(buf[:]) != "AAAAAAAAAA" {
		t.Errorf("Failed to read expected data. Expecting \"%s\", got \"%s\".",
			"AAAAAAAAAA", string(buf[:]))

		return
	}

	_, rErr = io.ReadFull(r, buf[:])

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if string(buf[:]) != "BBBBBBBBBB" {
		t.Errorf("Failed to read expected data. Expecting \"%s\", got \"%s\".",
			"BBBBBBBBBB", string(buf[:]))

		return
	}

	_, rErr = io.ReadFull(r, buf[:])

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if string(buf[:]) != "CCCCCCCCCC" {
		t.Errorf("Failed to read expected data. Expecting \"%s\", got \"%s\".",
			"CCCCCCCCCC", string(buf[:]))

		return
	}

	_, rErr = io.ReadFull(r, buf[:])

	if rErr != io.EOF {
		t.Error("Expecting read \"empty\" buffer will cause EOF error, got",
			rErr)

		return
	}
}

func TestByteSlice2(t *testing.T) {
	example := []byte("AAAAAAAAAABBBBBBBBBBCCCCCCCCCC")
	r := ByteSliceReader(example)

	buf := [1024]byte{}

	for {
		rLen, rErr := r.Read(buf[:])

		if rErr == io.EOF {
			break
		}

		if rErr != nil {
			t.Error("Failed to read due to error:", rErr)

			return
		}

		if string(buf[:rLen]) != string(example) {
			t.Errorf("Failed to read expected data. Expecting %d, got %d",
				example, buf[:rLen])

			return
		}
	}
}
