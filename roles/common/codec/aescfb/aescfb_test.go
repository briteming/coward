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
	"crypto/rand"
	"io"
	"sync"
	"testing"

	"github.com/reinit/coward/roles/common/codec/marker"
)

type dummyKey struct {
	Key []byte
}

func (d dummyKey) Get(size int) ([]byte, error) {
	result := make([]byte, size)

	copy(result, d.Key)

	return result, nil
}

type dummyMark struct{}

func (d dummyMark) Mark(marker.Mark) error {
	return nil
}

func TestAESCFB(t *testing.T) {
	k := dummyKey{
		Key: make([]byte, 64),
	}

	_, rErr := rand.Read(k.Key)

	if rErr != nil {
		t.Error("Failed to generate random key:", rErr)

		return
	}

	buf := bytes.NewBuffer(make([]byte, 0, 512))

	codec, codecErr := AESCFB(buf, k, 32, dummyMark{}, &sync.Mutex{})

	if codecErr != nil {
		t.Error("Failed to initialize codec:", codecErr)

		return
	}

	testData := make([]byte, 1024*64)

	_, rErr = rand.Read(testData)

	if rErr != nil {
		t.Error("Failed to generate random data:", rErr)

		return
	}

	expected := make([]byte, len(testData))

	copy(expected, testData)

	wLen, wErr := codec.Write(testData)

	if wErr != nil {
		t.Error("Failed to write data:", wErr)

		return
	}

	if wLen != len(testData) {
		t.Errorf("Invalid write length. Expecting %d, got %d",
			len(testData), wLen)

		return
	}

	resultData := make([]byte, len(testData))

	rLen, rErr := io.ReadFull(codec, resultData)

	if rErr != nil {
		t.Error("Failed to read data:", rErr)

		return
	}

	if rLen != len(resultData) {
		t.Errorf("Invalid read length. Expecting %d, got %d",
			len(resultData), rLen)

		return
	}

	if !bytes.Equal(resultData, expected) {
		t.Errorf("Reading invalid data. Expecting %d, got %d",
			expected, resultData)

		return
	}
}
