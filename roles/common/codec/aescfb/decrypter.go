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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"io"

	"github.com/reinit/coward/roles/common/codec/marker"
	"github.com/reinit/coward/roles/common/transceiver"
)

type decrypter struct {
	e *aescfb
	r io.Reader
}

func (a decrypter) Read(b []byte) (int, error) {
	if a.e.decryptBufReader.Len() > 0 {
		return a.e.decryptBufReader.Read(b)
	} else if a.e.decrypter == nil {
		iv := [aes.BlockSize]byte{}

		_, rErr := io.ReadFull(a.r, iv[:])

		if rErr != nil {
			return 0, rErr
		}

		a.e.decryptMarkerLock.Lock()
		markErr := a.e.decryptMarker.Mark(marker.Mark(iv[:]))
		a.e.decryptMarkerLock.Unlock()

		if markErr != nil {
			return 0, transceiver.WrapCodecError(markErr)
		}

		a.e.decrypter = cipher.NewCFBDecrypter(a.e.block, iv[:])
	}

	reader := decrypterReader{
		reader: a.r,
		stream: a.e.decrypter,
	}

	// Read front padding
	passErr := a.e.decryptPad.Passthrough(reader)

	if passErr != nil {
		return 0, passErr
	}

	// Read size
	sizeBuf := [2]byte{}

	_, rErr := io.ReadFull(reader, sizeBuf[:])

	if rErr != nil {
		return 0, rErr
	}

	size := 0
	size |= int(sizeBuf[0])
	size <<= 8
	size |= int(sizeBuf[1])

	if size > maxDataSegmentLength {
		return 0, ErrSegmentDataTooLong
	}

	// Record the HMAC of Size data
	_, wHMACErr := a.e.decryptHMAC.Write(sizeBuf[:])

	if wHMACErr != nil {
		return 0, transceiver.WrapCodecError(wHMACErr)
	}

	// Read data
	dReadBufLen := len(a.e.decryptBuf)

	if dReadBufLen < size {
		a.e.decryptBuf = make([]byte, size)
	}

	var rLen int

	rLen, rErr = io.ReadFull(reader, a.e.decryptBuf[:size])

	if rErr != nil {
		return 0, rErr
	}

	// Record the HMAC of Data
	_, wHMACErr = a.e.decryptHMAC.Write(a.e.decryptBuf[:size])

	if wHMACErr != nil {
		return 0, transceiver.WrapCodecError(wHMACErr)
	}

	// Read HMAC
	hmacValue := [hmacLength]byte{}

	_, rErr = io.ReadFull(reader, hmacValue[:])

	if rErr != nil {
		return 0, rErr
	}

	// Read tail padding
	passErr = a.e.decryptPad.Passthrough(reader)

	if passErr != nil {
		return 0, passErr
	}

	realHMACValue := [hmacLength]byte{}

	copy(realHMACValue[:], a.e.decryptHMAC.Sum(nil))

	if !hmac.Equal(hmacValue[:], realHMACValue[:]) {
		return 0, ErrSegmentDataVerificationFailed
	}

	// Record the HMAC data
	_, wHMACErr = a.e.decryptHMAC.Write(hmacValue[:])

	if wHMACErr != nil {
		return 0, transceiver.WrapCodecError(wHMACErr)
	}

	a.e.decryptBufReader = bytes.NewReader(a.e.decryptBuf[:rLen])

	return a.e.decryptBufReader.Read(b)
}
