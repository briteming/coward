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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/transceiver"
)

type encrypter struct {
	e *aescfb
	w io.Writer
}

func (a encrypter) getEncrypter() (cipher.Stream, error) {
	if a.e.encrypter == nil {
		iv := [aes.BlockSize]byte{}

		_, rErr := rand.Read(iv[:])

		if rErr != nil {
			return nil, rErr
		}

		_, wErr := rw.WriteFull(a.w, iv[:])

		if wErr != nil {
			return nil, wErr
		}

		a.e.encrypter = cipher.NewCFBEncrypter(a.e.block, iv[:])
	}

	return a.e.encrypter, nil
}

func (a encrypter) Write(b []byte) (int, error) {
	return a.WriteAll(b)
}

func (a encrypter) WriteAll(b ...[]byte) (int, error) {
	encrypter, encrypterInitErr := a.getEncrypter()

	if encrypterInitErr != nil {
		return 0, encrypterInitErr
	}

	writer := encrypterWriter{
		writer: a.w,
		stream: encrypter,
	}

	segmentWriter := rw.ByteSlicesWriter(func(size int, w io.Writer) error {
		// Write front padding, don't use w
		insertErr := a.e.encryptPad.Insert(writer)

		if insertErr != nil {
			return transceiver.WrapCodecError(insertErr)
		}

		sizeBuf := [2]byte{}
		sizeBuf[0] = byte(size >> 8)
		sizeBuf[1] = byte((size << 8) >> 8)

		_, wErr := rw.WriteFull(w, sizeBuf[:])

		if wErr != nil {
			return transceiver.WrapCodecError(wErr)
		}

		return wErr
	}, func(w io.Writer) error {
		currentHMAC := a.e.encryptHMAC.Sum(nil)

		_, wErr := rw.WriteFull(w, currentHMAC[:hmacLength])

		if wErr != nil {
			return transceiver.WrapCodecError(wErr)
		}

		// Tail padding, don't use w
		insertErr := a.e.encryptPad.Insert(writer)

		if insertErr != nil {
			return insertErr
		}

		return nil
	}, b...)

	totalWrite := 0

	for {
		// Write data
		wLen, wErr := segmentWriter.WriteMax(hashWriter{
			Writer: writer,
			hash:   a.e.encryptHMAC,
		}, maxDataSegmentLength)

		totalWrite += wLen

		if wErr != nil {
			return totalWrite, transceiver.WrapCodecError(wErr)
		}

		if segmentWriter.Remain() > 0 {
			continue
		}

		break
	}

	return totalWrite, nil
}
