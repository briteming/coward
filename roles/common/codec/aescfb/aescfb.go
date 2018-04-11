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
	"crypto/sha256"
	"hash"
	"io"
	"sync"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/codec/key"
	"github.com/reinit/coward/roles/common/codec/marker"
	"github.com/reinit/coward/roles/common/transceiver"
)

const (
	maxDataSegmentLength = 4096
	hmacLength           = 8
)

// Errors
var (
	ErrSegmentDataTooLong = transceiver.NewCodecError(
		"AES stream data segment was too long")

	ErrSegmentDataVerificationFailed = transceiver.NewCodecError(
		"AES stream data segment verification failed")
)

type aescfb struct {
	block             cipher.Block
	encrypter         cipher.Stream
	encryptHMAC       hash.Hash
	encryptPad        padding
	decrypter         cipher.Stream
	decryptHMAC       hash.Hash
	decryptPad        padding
	decryptBuf        []byte
	decryptBufReader  *bytes.Reader
	decryptMarker     marker.Marker
	decryptMarkerLock *sync.Mutex
}

// AESCFB returns a AES-CFB crypter
func AESCFB(
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

	return &aescfb{
		block:             blockCipher,
		encrypter:         nil,
		encryptHMAC:       hmac.New(sha256.New, keyValue),
		encryptPad:        padding{padBuf: [maxPaddingLength]byte{}},
		decrypter:         nil,
		decryptHMAC:       hmac.New(sha256.New, keyValue),
		decryptPad:        padding{padBuf: [maxPaddingLength]byte{}},
		decryptBuf:        nil,
		decryptBufReader:  bytes.NewReader(nil),
		decryptMarker:     mark,
		decryptMarkerLock: markLock,
	}, nil
}

func (a *aescfb) Encode(w io.Writer) rw.WriteWriteAll {
	return encrypter{e: a, w: w}
}

func (a *aescfb) Decode(r io.Reader) io.Reader {
	return decrypter{e: a, r: r}
}
