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
)

type byteSlice struct {
	buf    []byte
	remain int
}

type byteSlicesWriter struct {
	beforeWrite  func(size int, w io.Writer) error
	afterWrite   func(w io.Writer) error
	buf          [][]byte
	currentBuf   int
	currentByte  int
	remainLength int
}

type byteSlicesWriteOpt struct {
	Buf   int
	Start int
	End   int
}

// ByteSliceReader returns a reader of the input []byte
func ByteSliceReader(b []byte) io.Reader {
	return &byteSlice{
		buf:    b,
		remain: len(b),
	}
}

// ByteSlicesWriter returns a writer of the inputted []bytes
func ByteSlicesWriter(
	beforeWrite func(size int, w io.Writer) error,
	afterWrite func(w io.Writer) error,
	b ...[]byte,
) WriteMaxRemainer {
	totalLen := 0

	for _, bd := range b {
		totalLen += len(bd)
	}

	return &byteSlicesWriter{
		beforeWrite:  beforeWrite,
		afterWrite:   afterWrite,
		buf:          b,
		currentBuf:   0,
		currentByte:  0,
		remainLength: totalLen,
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

func (w *byteSlicesWriter) Remain() int {
	return w.remainLength
}

func (w *byteSlicesWriter) WriteMax(writer io.Writer, max int) (int, error) {
	if w.remainLength <= 0 {
		return 0, io.EOF
	}

	currentWritten := 0
	currentMax := max
	currentBuf := w.currentBuf
	currentByte := w.currentByte
	remainLength := w.remainLength

	if currentMax > w.remainLength {
		currentMax = w.remainLength
	}

	writeOpts := make([]byteSlicesWriteOpt, 0, (remainLength/currentMax)+1)

	for currentMax > currentWritten {
		bLen := len(w.buf[currentBuf][currentByte:])
		currentRemain := currentMax - currentWritten

		if bLen <= currentRemain {
			writeOpts = append(writeOpts, byteSlicesWriteOpt{
				Buf:   currentBuf,
				Start: currentByte,
				End:   currentByte + bLen,
			})

			currentWritten += bLen
			remainLength -= bLen

			currentByte = 0

			if currentBuf+1 >= len(w.buf) {
				break
			}

			currentBuf++

			continue
		}

		writeOpts = append(writeOpts, byteSlicesWriteOpt{
			Buf:   currentBuf,
			Start: currentByte,
			End:   currentByte + currentRemain,
		})

		currentWritten += currentRemain
		remainLength -= currentRemain
		currentByte += currentRemain
	}

	wbeErr := w.beforeWrite(currentWritten, writer)

	if wbeErr != nil {
		return 0, wbeErr
	}

	// Reuse this variable
	currentWritten = 0

	var writeErr error

	for _, opt := range writeOpts {
		wLen, wErr := WriteFull(writer, w.buf[opt.Buf][opt.Start:opt.End])

		currentWritten += wLen
		writeErr = wErr

		w.currentBuf = opt.Buf
		w.currentByte = opt.End
		w.remainLength -= wLen

		if writeErr != nil {
			break
		}
	}

	wbeErr = w.afterWrite(writer)

	if wbeErr != nil {
		return 0, wbeErr
	}

	return currentWritten, writeErr
}
