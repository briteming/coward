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

package resolve

import (
	"testing"
	"time"
)

func TestCached(t *testing.T) {
	cc := Cached(1*time.Second, 300*time.Millisecond, 1)

	results, resolveErr := cc.Resolve("localhost")

	if resolveErr != nil {
		t.Error("Failed to resolve due to error:", resolveErr)

		return
	}

	host, reverseErr := cc.Reverse(results[0])

	if reverseErr != nil {
		t.Error("Failed to reverse resolved result due to error:", reverseErr)

		return
	}

	if host != "localhost" {
		t.Errorf("Expecting reversed resolved result to be \"localhost\", "+
			"got %s", host)

		return
	}

	_, resolveErr = cc.Resolve("www.wikipedia.com")

	if resolveErr != ErrCachedCacheTooMany {
		t.Error(
			"Cache too many resolved result must cause ErrCachedCacheTooMany")

		return
	}
}
