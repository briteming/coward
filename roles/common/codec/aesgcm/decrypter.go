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
	"bytes"
	"io"

	"github.com/reinit/coward/roles/common/codec/marker"
	"github.com/reinit/coward/roles/common/transceiver"
)

type decrypter struct {
	e *aesgcm
	r io.Reader
}

func (a decrypter) Read(b []byte) (int, error) {
	if a.e.decryptReader.Len() > 0 {
		return a.e.decryptReader.Read(b)
	} else if !a.e.decrypterInited {
		_, rErr := io.ReadFull(a.r, a.e.decryptNonceBuf[:])

		if rErr != nil {
			return 0, rErr
		}

		a.e.decryptMarkerLock.Lock()
		markErr := a.e.decryptMarker.Mark(marker.Mark(a.e.decryptNonceBuf[:]))
		a.e.decryptMarkerLock.Unlock()

		if markErr != nil {
			return 0, transceiver.WrapCodecError(markErr)
		}

		a.e.decrypterInited = true
	}

	rBuf := a.e.getDecryptBuf(3)

	_, rErr := io.ReadFull(a.r, rBuf)

	if rErr != nil {
		return 0, rErr
	}

	sizeData, sizeDataOpenErr := a.e.decrypter.Open(
		nil, a.e.decryptNonceBuf[:], rBuf, nil)

	if sizeDataOpenErr != nil {
		return 0, transceiver.WrapCodecError(sizeDataOpenErr)
	}

	if len(sizeData) != 3 {
		return 0, ErrInvalidSizeDataLength
	}

	a.e.nonceIncreament(a.e.decryptNonceBuf[:])

	size := 0

	size |= int(sizeData[0])
	size <<= 8
	size |= int(sizeData[1])

	if size > maxDataBlockSize {
		return 0, ErrDataBlockTooLarge
	}

	// Got some padding to read?
	if sizeData[2] > 0 {
		if sizeData[2] > maxPaddingBlockSize {
			return 0, ErrPaddingBlockTooLarge
		}

		rBuf = a.e.getDecryptBuf(int(sizeData[2]))

		_, rErr = io.ReadFull(a.r, rBuf)

		if rErr != nil {
			return 0, rErr
		}

		_, paddingOpenErr := a.e.decrypter.Open(
			nil, a.e.decryptNonceBuf[:], rBuf, nil)

		if paddingOpenErr != nil {
			return 0, transceiver.WrapCodecError(paddingOpenErr)
		}

		a.e.nonceIncreament(a.e.decryptNonceBuf[:])
	}

	rBuf = a.e.getDecryptBuf(size)

	_, rErr = io.ReadFull(a.r, rBuf)

	if rErr != nil {
		return 0, rErr
	}

	dataData, dataOpenErr := a.e.decrypter.Open(
		nil, a.e.decryptNonceBuf[:], rBuf, nil)

	if dataOpenErr != nil {
		return 0, transceiver.WrapCodecError(dataOpenErr)
	}

	a.e.nonceIncreament(a.e.decryptNonceBuf[:])

	a.e.decryptReader = bytes.NewReader(dataData)

	return a.e.decryptReader.Read(b)
}
