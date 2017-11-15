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

package aesgcm

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"sync"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/codec/key"
	"github.com/reinit/coward/roles/common/codec/marker"
	"github.com/reinit/coward/roles/common/transceiver"
)

const (
	nonceSize           = 12
	maxDataBlockSize    = 4096
	maxPaddingBlockSize = 32
)

// Errors
var (
	ErrDataBlockTooLarge = transceiver.NewCodecError(
		"AES-GCM Data block too large, decode refused")

	ErrPaddingBlockTooLarge = transceiver.NewCodecError(
		"AES-GCM Padding block too large, decode refused")

	ErrInvalidSizeDataLength = transceiver.NewCodecError(
		"The length information of size data is invalid")
)

type aesgcm struct {
	rw                   io.ReadWriter
	block                cipher.Block
	encrypter            cipher.AEAD
	encrypterInited      bool
	encrypterNonceBuf    [nonceSize]byte
	encrypterPaddingBuf  [maxPaddingBlockSize]byte
	decrypter            cipher.AEAD
	decrypterInited      bool
	decryptBuf           []byte
	decryptCipherTextBuf []byte
	decryptReader        *bytes.Reader
	decryptNonceBuf      [nonceSize]byte
	decryptMarker        marker.Marker
	decryptMarkerLock    *sync.Mutex
}

// AESGCM returns a AES-GCM crypter
func AESGCM(
	conn io.ReadWriter,
	kg key.Key,
	keySize int,
	mark marker.Marker,
	markLock *sync.Mutex,
) (io.ReadWriter, error) {
	keyValue, keyErr := kg.Get(keySize)

	if keyErr != nil {
		return nil, keyErr
	}

	blockCipher, blockCipherErr := aes.NewCipher(keyValue)

	if blockCipherErr != nil {
		return nil, blockCipherErr
	}

	gcmEncrypter, gcmEncrypterErr := cipher.NewGCMWithNonceSize(
		blockCipher, nonceSize)

	if gcmEncrypterErr != nil {
		return nil, gcmEncrypterErr
	}

	gcmDecrypter, gcmDecrypterErr := cipher.NewGCMWithNonceSize(
		blockCipher, nonceSize)

	if gcmDecrypterErr != nil {
		return nil, gcmDecrypterErr
	}

	return &aesgcm{
		rw:                   conn,
		block:                blockCipher,
		encrypter:            gcmEncrypter,
		encrypterInited:      false,
		encrypterNonceBuf:    [nonceSize]byte{},
		encrypterPaddingBuf:  [maxPaddingBlockSize]byte{},
		decrypter:            gcmDecrypter,
		decrypterInited:      false,
		decryptBuf:           nil,
		decryptCipherTextBuf: nil,
		decryptReader:        bytes.NewReader(nil),
		decryptNonceBuf:      [nonceSize]byte{},
		decryptMarker:        mark,
		decryptMarkerLock:    markLock,
	}, nil
}

func (a *aesgcm) nonceIncreament(nonce []byte) {
	// Do a increament in reversed byte order
	for nIdx := range nonce {
		if nonce[nIdx] < 255 {
			nonce[nIdx]++

			break
		}

		nonce[nIdx] = 0
	}
}

func (a *aesgcm) Read(b []byte) (int, error) {
	if a.decryptReader.Len() > 0 {
		return a.decryptReader.Read(b)
	} else if !a.decrypterInited {
		_, rErr := io.ReadFull(a.rw, a.decryptNonceBuf[:])

		if rErr != nil {
			return 0, rErr
		}

		a.decryptMarkerLock.Lock()
		markErr := a.decryptMarker.Mark(marker.Mark(a.decryptNonceBuf[:]))
		a.decryptMarkerLock.Unlock()

		if markErr != nil {
			return 0, markErr
		}

		a.decrypterInited = true
	}

	sizeCipherTextReadLen := a.decrypter.Overhead() + 3

	if len(a.decryptCipherTextBuf) < sizeCipherTextReadLen {
		a.decryptCipherTextBuf = make([]byte, sizeCipherTextReadLen)
	}

	_, rErr := io.ReadFull(
		a.rw, a.decryptCipherTextBuf[:sizeCipherTextReadLen])

	if rErr != nil {
		return 0, rErr
	}

	sizeData, sizeDataOpenErr := a.decrypter.Open(
		nil,
		a.decryptNonceBuf[:],
		a.decryptCipherTextBuf[:sizeCipherTextReadLen],
		nil)

	if sizeDataOpenErr != nil {
		return 0, transceiver.WrapCodecError(sizeDataOpenErr)
	}

	if len(sizeData) != 3 {
		return 0, ErrInvalidSizeDataLength
	}

	a.nonceIncreament(a.decryptNonceBuf[:])

	size := uint16(0)

	size |= uint16(sizeData[0])
	size <<= 8
	size |= uint16(sizeData[1])

	if size > maxDataBlockSize {
		return 0, ErrDataBlockTooLarge
	}

	// Got some padding to read?
	if sizeData[2] > 0 {
		if sizeData[2] > maxPaddingBlockSize {
			return 0, ErrPaddingBlockTooLarge
		}

		paddingReadLen := a.decrypter.Overhead() + int(sizeData[2])

		if len(a.decryptCipherTextBuf) < paddingReadLen {
			a.decryptCipherTextBuf = make([]byte, paddingReadLen)
		}

		_, rErr = io.ReadFull(a.rw, a.decryptCipherTextBuf[:paddingReadLen])

		if rErr != nil {
			return 0, rErr
		}

		_, paddingOpenErr := a.decrypter.Open(
			nil,
			a.decryptNonceBuf[:],
			a.decryptCipherTextBuf[:paddingReadLen],
			nil)

		if paddingOpenErr != nil {
			return 0, paddingOpenErr
		}

		a.nonceIncreament(a.decryptNonceBuf[:])
	}

	actualCipherTextReadLen := a.decrypter.Overhead() + int(size)

	if len(a.decryptCipherTextBuf) < actualCipherTextReadLen {
		a.decryptCipherTextBuf = make([]byte, actualCipherTextReadLen)
	}

	_, rErr = io.ReadFull(
		a.rw, a.decryptCipherTextBuf[:actualCipherTextReadLen])

	if rErr != nil {
		return 0, rErr
	}

	dataData, dataOpenErr := a.decrypter.Open(
		nil,
		a.decryptNonceBuf[:],
		a.decryptCipherTextBuf[:actualCipherTextReadLen],
		nil)

	if dataOpenErr != nil {
		return 0, transceiver.WrapCodecError(dataOpenErr)
	}

	a.nonceIncreament(a.decryptNonceBuf[:])

	a.decryptReader = bytes.NewReader(dataData)

	return a.decryptReader.Read(b)
}

func (a *aesgcm) Write(b []byte) (int, error) {
	if !a.encrypterInited {
		_, rErr := rand.Read(a.encrypterNonceBuf[:])

		if rErr != nil {
			return 0, rErr
		}

		_, wErr := rw.WriteFull(a.rw, a.encrypterNonceBuf[:])

		if wErr != nil {
			return 0, wErr
		}

		a.encrypterInited = true
	}

	bLen := len(b)
	start := 0
	sizeBuf := [3]byte{}

	_, paddingSizeReadErr := rand.Read(sizeBuf[2:])

	if paddingSizeReadErr != nil {
		return 0, paddingSizeReadErr
	}

	sizeBuf[2] %= maxPaddingBlockSize

	for start < bLen {
		end := start + maxDataBlockSize

		if end > bLen {
			end = bLen
		}

		writeLen := uint16(end - start)
		sizeBuf[0] = byte(uint16(writeLen) >> 8)
		sizeBuf[1] = byte((uint16(writeLen) << 8) >> 8)

		_, wErr := rw.WriteFull(a.rw, a.encrypter.Seal(
			nil, a.encrypterNonceBuf[:], sizeBuf[:], nil))

		if wErr != nil {
			return start, wErr
		}

		a.nonceIncreament(a.encrypterNonceBuf[:])

		// Notice the padding may not apply to the data block at all
		if sizeBuf[2] > 0 {
			_, rErr := rand.Read(a.encrypterPaddingBuf[:sizeBuf[2]])

			if rErr != nil {
				return start, rErr
			}

			_, wErr = rw.WriteFull(a.rw, a.encrypter.Seal(
				nil, a.encrypterNonceBuf[:],
				a.encrypterPaddingBuf[:sizeBuf[2]],
				nil))

			if wErr != nil {
				return start, wErr
			}

			if a.encrypterPaddingBuf[0] > 0 {
				sizeBuf[2] %= a.encrypterPaddingBuf[0]
			} else {
				sizeBuf[2] = 0
			}

			a.nonceIncreament(a.encrypterNonceBuf[:])
		} else {
			_, paddingSizeReadErr := rand.Read(sizeBuf[2:])

			if paddingSizeReadErr != nil {
				return 0, paddingSizeReadErr
			}

			sizeBuf[2] %= maxPaddingBlockSize
		}

		_, wErr = rw.WriteFull(a.rw, a.encrypter.Seal(
			nil, a.encrypterNonceBuf[:], b[start:end], nil))

		if wErr != nil {
			return start, wErr
		}

		start = end

		a.nonceIncreament(a.encrypterNonceBuf[:])
	}

	return start, nil
}
