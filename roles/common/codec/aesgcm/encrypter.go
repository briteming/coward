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

package aesgcm

import (
	"crypto/rand"
	"io"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/transceiver"
)

type encrypter struct {
	e *aesgcm
	w io.Writer
}

func (a encrypter) Write(b []byte) (int, error) {
	return a.WriteAll(b)
}

func (a encrypter) WriteAll(b ...[]byte) (int, error) {
	if !a.e.encrypterInited {
		_, rErr := rand.Read(a.e.encrypterNonceBuf[:])

		if rErr != nil {
			return 0, transceiver.WrapCodecError(rErr)
		}

		_, wErr := rw.WriteFull(a.w, a.e.encrypterNonceBuf[:])

		if wErr != nil {
			return 0, wErr
		}

		a.e.encrypterInited = true
	}

	segmentWriter := rw.ByteSlicesWriter(func(size int, w io.Writer) error {
		// Write header
		// NOTICE: we didn't use w here
		sizePadBuf := [16]byte{}

		_, rErr := rand.Read(sizePadBuf[2:])

		if rErr != nil {
			return rErr
		}

		sizePadBuf[0] = byte(size >> 8)
		sizePadBuf[1] = byte((size << 8) >> 8)
		sizePadBuf[2] %= maxPaddingBlockSize - 3

		_, wErr := rw.WriteFull(a.w, a.e.encrypter.Seal(
			nil, a.e.encrypterNonceBuf[:],
			sizePadBuf[:3],
			nil))

		if wErr != nil {
			return wErr
		}

		a.e.nonceIncreament(a.e.encrypterNonceBuf[:])

		if sizePadBuf[2] > 0 {
			_, wErr = rw.WriteFull(a.w, a.e.encrypter.Seal(
				nil, a.e.encrypterNonceBuf[:],
				sizePadBuf[3:3+sizePadBuf[2]],
				nil))

			if wErr != nil {
				return wErr
			}

			a.e.nonceIncreament(a.e.encrypterNonceBuf[:])
		}

		return nil
	}, func(w io.Writer) error {
		return nil
	}, b...)

	totalWritten := 0

	for {
		// Write data
		wLen, wErr := segmentWriter.WriteMax(a.e.encryptBuf, maxDataBlockSize)

		totalWritten += wLen

		if wErr != nil {
			return totalWritten, transceiver.WrapCodecError(wErr)
		}

		_, wErr = rw.WriteFull(a.w, a.e.encrypter.Seal(
			nil, a.e.encrypterNonceBuf[:],
			a.e.encryptBuf.Bytes(),
			nil))

		a.e.encryptBuf.Reset()

		if wErr != nil {
			return totalWritten, wErr
		}

		a.e.nonceIncreament(a.e.encrypterNonceBuf[:])

		if segmentWriter.Remain() > 0 {
			continue
		}

		break
	}

	return totalWritten, nil
}
