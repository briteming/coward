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
	"bytes"
	"testing"
	"time"
)

func TestTimed(t *testing.T) {
	testTime := time.Time{}
	tt := Timed([]byte("Hello World"), 1*time.Second, func() time.Time {
		return testTime
	})

	firstKey, _ := tt.Get(32)

	resultKey, _ := tt.Get(32)
	if !bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at same time period must remain the same")

		return
	}

	testTime = testTime.Add(500 * time.Millisecond)

	resultKey, _ = tt.Get(32)
	if !bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at same time period must remain the same")

		return
	}

	testTime = testTime.Add(400 * time.Millisecond)

	resultKey, _ = tt.Get(32)
	if !bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at same time period must remain the same")

		return
	}

	testTime = testTime.Add(100 * time.Millisecond)

	resultKey, _ = tt.Get(32)
	if bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at different time period must be different")

		return
	}
}
