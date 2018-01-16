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

package connection

import (
	"bytes"
	"io"
	"testing"
)

type dummyCodec struct {
	io.ReadWriter
}

func (d dummyCodec) Read(b []byte) (int, error) {
	rLen, rErr := d.ReadWriter.Read(b)

	if rErr != nil {
		return rLen, rErr
	}

	for v := range b {
		b[v] ^= 66
	}

	return rLen, nil
}

func (d dummyCodec) Write(b []byte) (int, error) {
	for v := range b {
		b[v] ^= 66
	}

	return d.ReadWriter.Write(b)
}

func (d dummyCodec) Commit() error {
	return nil
}

func TestCodec(t *testing.T) {
	d := &dummyConnection{
		buf: bytes.NewBuffer(make([]byte, 0, 4096)),
	}
	c, _ := Codec(d, func(n io.ReadWriter) (io.ReadWriter, error) {
		return dummyCodec{
			ReadWriter: n,
		}, nil
	})

	wLen, wErr := c.Write([]byte("Hello World"))

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != 11 {
		t.Errorf("Failed to write correct length, expected 11, got %d", wLen)

		return
	}

	buf := [1024]byte{}

	rLen, rErr := c.Read(buf[:])

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if rLen != 11 {
		t.Errorf("Failed to read expected length. Expecting 11, got %d", rLen)

		return
	}

	if !bytes.Equal(buf[:rLen], []byte("Hello World")) {
		t.Errorf("Failed to read expected data. Expecting \"Hello World\", "+
			"got %s", buf[:rLen])

		return
	}
}
