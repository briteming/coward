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

package aescfb

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"hash"
	"io"
	"math"
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

type aescfg struct {
	rw                io.ReadWriter
	block             cipher.Block
	encrypter         io.Writer
	encryptHMAC       hash.Hash
	encryptPad        padding
	decrypter         io.Reader
	decryptHMAC       hash.Hash
	decryptPad        padding
	decryptBuf        []byte
	decryptBufReader  *bytes.Reader
	decryptMarker     marker.Marker
	decryptMarkerLock *sync.Mutex
}

// AESCFB returns a AES-CFB crypter
func AESCFB(
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

	return &aescfg{
		rw:                conn,
		block:             blockCipher,
		encrypter:         nil,
		encryptHMAC:       hmac.New(sha256.New, keyValue),
		encryptPad:        padding{nextPaddingLength: 0},
		decrypter:         nil,
		decryptHMAC:       hmac.New(sha256.New, keyValue),
		decryptPad:        padding{nextPaddingLength: 0},
		decryptBuf:        nil,
		decryptBufReader:  bytes.NewReader(nil),
		decryptMarker:     mark,
		decryptMarkerLock: markLock,
	}, nil
}

func (a *aescfg) hmac(m hash.Hash, b []byte, out []byte) (int, error) {
	_, wErr := m.Write(b)

	if wErr != nil {
		return 0, wErr
	}

	return copy(out, m.Sum(nil)), nil
}

func (a *aescfg) Read(b []byte) (int, error) {
	if a.decryptBufReader.Len() > 0 {
		return a.decryptBufReader.Read(b)
	} else if a.decrypter == nil {
		iv := [aes.BlockSize]byte{}

		_, rErr := io.ReadFull(a.rw, iv[:])

		if rErr != nil {
			return 0, rErr
		}

		a.decryptMarkerLock.Lock()
		markErr := a.decryptMarker.Mark(marker.Mark(iv[:]))
		a.decryptMarkerLock.Unlock()

		if markErr != nil {
			return 0, markErr
		}

		a.decrypter = decrypter{
			reader: a.rw,
			stream: cipher.NewCFBDecrypter(a.block, iv[:]),
		}
	}

	// Read front padding
	passErr := a.decryptPad.Passthrough(a.decrypter)

	if passErr != nil {
		return 0, passErr
	}

	// Read hmac
	hmacValue := [hmacLength]byte{}

	_, rErr := io.ReadFull(a.decrypter, hmacValue[:])

	if rErr != nil {
		return 0, rErr
	}

	// Read size
	sizeBuf := [2]byte{}

	_, rErr = io.ReadFull(a.decrypter, sizeBuf[:])

	if rErr != nil {
		return 0, rErr
	}

	size := uint16(0)
	size |= uint16(sizeBuf[0])
	size <<= 8
	size |= uint16(sizeBuf[1])

	if size > maxDataSegmentLength {
		return 0, ErrSegmentDataTooLong
	}

	// Read data
	dReadBufLen := len(a.decryptBuf)

	if dReadBufLen < int(size) {
		a.decryptBuf = make([]byte, size)
	}

	var rLen int

	rLen, rErr = io.ReadFull(a.decrypter, a.decryptBuf[:size])

	if rErr != nil {
		return 0, rErr
	}

	// Read tail padding
	passErr = a.decryptPad.Passthrough(a.decrypter)

	if passErr != nil {
		return 0, passErr
	}

	// Verify
	_, wHMACSizeErr := a.decryptHMAC.Write( // Write Size in first
		sizeBuf[:]) // so along the way we can verify that as well

	if wHMACSizeErr != nil {
		return 0, wHMACSizeErr
	}

	realHMACValue := [hmacLength]byte{}

	_, realHMACErr := a.hmac(
		a.decryptHMAC, a.decryptBuf[:rLen], realHMACValue[:])

	if realHMACErr != nil {
		return 0, realHMACErr
	}

	if !hmac.Equal(hmacValue[:], realHMACValue[:]) {
		return 0, ErrSegmentDataVerificationFailed
	}

	a.decryptBufReader = bytes.NewReader(a.decryptBuf[:rLen])

	return a.decryptBufReader.Read(b)
}

func (a *aescfg) Write(b []byte) (int, error) {
	if a.encrypter == nil {
		iv := [aes.BlockSize]byte{}

		_, rErr := rand.Read(iv[:])

		if rErr != nil {
			return 0, rErr
		}

		a.encrypter = encrypter{
			writer: a.rw,
			stream: cipher.NewCFBEncrypter(a.block, iv[:]),
		}

		_, wErr := rw.WriteFull(a.rw, iv[:])

		if wErr != nil {
			return 0, wErr
		}
	}

	bLen := len(b)
	hmacBuf := [hmacLength]byte{}
	wStart := 0
	maxPaddingLen := bLen
	sizeBuf := [2]byte{}

	if bLen > 256 {
		maxPaddingLen = (bLen / 10) % math.MaxUint8
	}

	for wStart < bLen {
		wEnd := wStart + maxDataSegmentLength

		if wEnd > bLen {
			wEnd = bLen
		}

		// Write front padding
		insertErr := a.encryptPad.Insert(a.encrypter, byte(maxPaddingLen))

		if insertErr != nil {
			return wStart, insertErr
		}

		size := uint16(wEnd - wStart)

		sizeBuf[0] = byte(size >> 8)
		sizeBuf[1] = byte((size << 8) >> 8)

		_, wHMACErr := a.encryptHMAC.Write(sizeBuf[:])

		if wHMACErr != nil {
			return wStart, wHMACErr
		}

		_, hmacErr := a.hmac(a.encryptHMAC, b[wStart:wEnd], hmacBuf[:])

		if hmacErr != nil {
			return wStart, hmacErr
		}

		_, wErr := rw.WriteFull(a.encrypter, hmacBuf[:])

		if wErr != nil {
			return wStart, wErr
		}

		// Write size
		_, wErr = rw.WriteFull(a.encrypter, sizeBuf[:])

		if wErr != nil {
			return wStart, wErr
		}

		// Write data
		_, wErr = rw.WriteFull(a.encrypter, b[wStart:wEnd])

		if wErr != nil {
			return wStart, wErr
		}

		// Write tail padding
		insertErr = a.encryptPad.Insert(a.encrypter, byte(maxPaddingLen))

		if insertErr != nil {
			return wStart, insertErr
		}

		wStart = wEnd
	}

	return wStart, nil
}
