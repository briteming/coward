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

package aescfb

import (
	"bytes"
	"io"
	"testing"
)

func TestPadding(t *testing.T) {
	data := bytes.NewBuffer(make([]byte, 0, 1024))
	rPad := padding{
		nextPaddingLength: 0,
	}
	wPad := padding{
		nextPaddingLength: 0,
	}

	insertErr := wPad.Insert(data, 128)

	if insertErr != nil {
		t.Error("Padding insert failed:", insertErr)

		return
	}

	data.Write([]byte("Test data segment 1"))

	insertErr = wPad.Insert(data, 128)

	if insertErr != nil {
		t.Error("Padding insert failed:", insertErr)

		return
	}

	data.Write([]byte("Test data segment 2"))

	insertErr = wPad.Insert(data, 128)

	if insertErr != nil {
		t.Error("Padding insert failed:", insertErr)

		return
	}

	data.Write([]byte("Test data segment 3"))

	// Start pass
	passErr := rPad.Passthrough(data)

	if passErr != nil {
		t.Error("Padding pass failed:", passErr)

		return
	}

	dRead := [19]byte{}

	_, rErr := io.ReadFull(data, dRead[:])

	if rErr != nil {
		t.Error("Padding data read error:", rErr)

		return
	}

	if !bytes.Equal(dRead[:], []byte("Test data segment 1")) {
		t.Errorf("Expecting first data segment would be %d, got %d",
			[]byte("Test data segment 1"), dRead[:])

		return
	}

	passErr = rPad.Passthrough(data)

	if passErr != nil {
		t.Error("Padding pass failed:", passErr)

		return
	}

	_, rErr = io.ReadFull(data, dRead[:])

	if rErr != nil {
		t.Error("Padding data read error:", rErr)

		return
	}

	if !bytes.Equal(dRead[:], []byte("Test data segment 2")) {
		t.Errorf("Expecting first data segment would be %d, got %d",
			[]byte("Test data segment 2"), dRead[:])

		return
	}

	passErr = rPad.Passthrough(data)

	if passErr != nil {
		t.Error("Padding pass failed:", passErr)

		return
	}

	_, rErr = io.ReadFull(data, dRead[:])

	if rErr != nil {
		t.Error("Padding data read error:", rErr)

		return
	}

	if !bytes.Equal(dRead[:], []byte("Test data segment 3")) {
		t.Errorf("Expecting first data segment would be %d, got %d",
			[]byte("Test data segment 3"), dRead[:])

		return
	}
}
