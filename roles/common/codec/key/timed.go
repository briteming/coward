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

package key

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

type timed struct {
	k [sha256.Size]byte
	d time.Duration
	t func() time.Time
}

// Timed returns a timed key generater
func Timed(k []byte, d time.Duration, timer func() time.Time) Key {
	return timed{
		k: sha256.Sum256(k),
		d: d,
		t: timer,
	}
}

func (t timed) Get(size int) ([]byte, error) {
	hasher := hmac.New(sha256.New, t.k[:])
	nowByte := [8]byte{}
	nowInt := uint64(t.t().Truncate(t.d).Unix())

	binary.BigEndian.PutUint64(nowByte[:], nowInt)

	_, wErr := hasher.Write(nowByte[:])

	if wErr != nil {
		return nil, wErr
	}

	result := make([]byte, size)

	copy(result, hasher.Sum(nil))

	return result, nil
}
