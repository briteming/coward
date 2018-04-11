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
	"bytes"
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

func TestByteSlicesWriter(t *testing.T) {
	lastSize := 0
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	w := ByteSlicesWriter(func(s int, w io.Writer) error {
		lastSize = s

		return nil
	}, func(w io.Writer) error {
		return nil
	}, []byte("Hello"), []byte("World"), []byte("!"))

	wLen, wErr := w.WriteMax(b, 3)

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != 3 {
		t.Errorf("Failed to write %d bytes, got %d", 3, wLen)

		return
	}

	if lastSize != wLen {
		t.Errorf("Notified to write %d bytes, actually written %d bytes",
			lastSize, wLen)

		return
	}

	if !bytes.Equal(b.Bytes(), []byte("Hel")) {
		t.Errorf("Failed to write %d, got %d", []byte("Hel"), b.Bytes())

		return
	}

	b.Reset()

	wLen, wErr = w.WriteMax(b, 4)

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != 4 {
		t.Errorf("Failed to write %d bytes, got %d", 4, wLen)

		return
	}

	if lastSize != wLen {
		t.Errorf("Notified to write %d bytes, actually written %d bytes",
			lastSize, wLen)

		return
	}

	if !bytes.Equal(b.Bytes(), []byte("loWo")) {
		t.Errorf("Failed to write %d, got %d", []byte("loWo"), b.Bytes())

		return
	}

	b.Reset()

	wLen, wErr = w.WriteMax(b, 3)

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != 3 {
		t.Errorf("Failed to write %d bytes, got %d", 3, wLen)

		return
	}

	if lastSize != wLen {
		t.Errorf("Notified to write %d bytes, actually written %d bytes",
			lastSize, wLen)

		return
	}

	if !bytes.Equal(b.Bytes(), []byte("rld")) {
		t.Errorf("Failed to write %d, got %d", []byte("rld"), b.Bytes())

		return
	}

	b.Reset()

	wLen, wErr = w.WriteMax(b, 3)

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != 1 {
		t.Errorf("Failed to write %d bytes, got %d", 1, wLen)

		return
	}

	if lastSize != wLen {
		t.Errorf("Notified to write %d bytes, actually written %d bytes",
			lastSize, wLen)

		return
	}

	if !bytes.Equal(b.Bytes(), []byte("!")) {
		t.Errorf("Failed to write %d, got %d", []byte("!"), b.Bytes())

		return
	}

	b.Reset()
}

func TestByteSlicesWriter2(t *testing.T) {
	lastSize := 0
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	w := ByteSlicesWriter(func(s int, w io.Writer) error {
		lastSize = s

		return nil
	}, func(w io.Writer) error {
		return nil
	}, []byte("Hello"), []byte("World"), []byte("!"))

	wLen, wErr := w.WriteMax(b, 16)

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != 11 {
		t.Errorf("Failed to write %d bytes, got %d", 11, wLen)

		return
	}

	if lastSize != wLen {
		t.Errorf("Notified to write %d bytes, actually written %d bytes",
			lastSize, wLen)

		return
	}

	if !bytes.Equal(b.Bytes(), []byte("HelloWorld!")) {
		t.Errorf("Failed to write %d, got %d", []byte("Hel"), b.Bytes())

		return
	}

	b.Reset()
}

func BenchmarkByteSlicesWriter(b *testing.B) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := ByteSlicesWriter(func(s int, w io.Writer) error {
			return nil
		}, func(w io.Writer) error {
			return nil
		}, []byte("Hello"), []byte("World"), []byte("!"))

		for {
			_, wErr := w.WriteMax(buf, 3)

			if wErr == nil {
				continue
			}

			if w.Remain() <= 0 {
				break
			}

			b.Error("Failed to write due to error:", wErr)
		}

		if !bytes.Equal(buf.Bytes(), []byte("HelloWorld!")) {
			b.Error("Failed to write expected bytes")

			return
		}

		buf.Reset()
	}
}
