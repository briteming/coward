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
	"testing"
)

func TestPrefixerWrite(t *testing.T) {
	reqPrefix := []byte("Hello")
	expectedRespPrefix := []byte("Oh Hi")
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	www := New(buf, reqPrefix, expectedRespPrefix)

	wLen, wErr := www.Write([]byte("Test"))

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != 4 {
		t.Errorf("Expecting 4 bytes been written, got %d", wLen)

		return
	}

	if !bytes.Equal(buf.Bytes(), []byte("HelloTest")) {
		t.Errorf("Failed to write expected data, expecting %d, got %d",
			[]byte("HelloTest"), buf.Bytes())

		return
	}
}

func TestPrefixerRead(t *testing.T) {
	reqPrefix := []byte("Oh Hi")
	expectedRespPrefix := []byte("Hello")
	buf := bytes.NewBuffer([]byte("HelloTest"))
	www := New(buf, reqPrefix, expectedRespPrefix)

	rBuf := [4]byte{}

	rLen, rErr := www.Read(rBuf[:])

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if rLen != 4 {
		t.Errorf("Expecting 4 bytes to be read, got %d", rLen)

		return
	}

	if !bytes.Equal(rBuf[:], []byte("Test")) {
		t.Errorf("Failed to read expected data. Expecting to read %d, got %d",
			[]byte("Test"), rBuf[:])

		return
	}
}

func TestPrefixerRead2(t *testing.T) {
	reqPrefix := []byte("Oh Hi")
	expectedRespPrefix := bytes.Repeat([]byte("Hello"), 1024)

	byteData := make([]byte, len(expectedRespPrefix)+4)

	copy(byteData[:len(expectedRespPrefix)], expectedRespPrefix)
	copy(byteData[len(expectedRespPrefix):], []byte("Test"))

	buf := bytes.NewBuffer(byteData)
	www := New(buf, reqPrefix, expectedRespPrefix)

	rBuf := [4]byte{}

	rLen, rErr := www.Read(rBuf[:])

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if rLen != 4 {
		t.Errorf("Expecting 4 bytes to be read, got %d", rLen)

		return
	}

	if !bytes.Equal(rBuf[:], []byte("Test")) {
		t.Errorf("Failed to read expected data. Expecting to read %d, got %d",
			[]byte("Test"), rBuf[:])

		return
	}
}
