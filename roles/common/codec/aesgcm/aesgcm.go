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
	"crypto/aes"
	"crypto/cipher"
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
	maxPaddingBlockSize = 16
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
	block                cipher.Block
	encrypter            cipher.AEAD
	encrypterInited      bool
	encrypterNonceBuf    [nonceSize]byte
	encrypterPaddingBuf  [maxPaddingBlockSize]byte
	encryptBuf           *bytes.Buffer
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
	kg key.Key,
	keySize int,
	mark marker.Marker,
	markLock *sync.Mutex,
) (rw.Codec, error) {
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
		block:                blockCipher,
		encrypter:            gcmEncrypter,
		encrypterInited:      false,
		encrypterNonceBuf:    [nonceSize]byte{},
		encrypterPaddingBuf:  [maxPaddingBlockSize]byte{},
		encryptBuf:           bytes.NewBuffer(nil),
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

func (a *aesgcm) getDecryptBuf(size int) []byte {
	sizeCipherTextReadLen := a.decrypter.Overhead() + size

	if len(a.decryptCipherTextBuf) < sizeCipherTextReadLen {
		a.decryptCipherTextBuf = make([]byte, sizeCipherTextReadLen)
	}

	return a.decryptCipherTextBuf[:sizeCipherTextReadLen]
}

func (a *aesgcm) Encode(w io.Writer) rw.WriteWriteAll {
	return encrypter{
		e: a,
		w: w,
	}
}

func (a *aesgcm) Decode(r io.Reader) io.Reader {
	return decrypter{
		e: a,
		r: r,
	}
}
