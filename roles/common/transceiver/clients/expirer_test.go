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

package clients

import (
	"strconv"
	"testing"

	"github.com/reinit/coward/roles/common/transceiver"
)

func TestExpirerExpire(t *testing.T) {
	e := expirer{
		dests:   make([]transceiver.Destination, 8),
		nextIdx: 0,
		maxSize: 8,
	}

	var lastIdx int

	for i := 0; i <= 7; i++ {
		bumped, addedIdx := e.Add(
			transceiver.Destination(strconv.FormatInt(int64(i), 10)))

		if bumped != "" {
			t.Errorf("Unexpected bumping when trying to add %d items", i+1)
		}

		lastIdx = addedIdx
	}

	if lastIdx != 7 {
		t.Errorf("Expecting the last added item will be at index %d, got %d",
			7, lastIdx)

		return
	}

	bumped, _ := e.Add(transceiver.Destination(strconv.FormatInt(8, 10)))

	if bumped != "0" {
		t.Errorf("Expecting item %s got bumped out after Add, got %s instead",
			"0", bumped)

		return
	}

	bumped, _ = e.Bump(lastIdx)

	if bumped != "1" {
		t.Errorf("Expecting item %s got bumped out after Bump, got %s instead",
			"1", bumped)

		return
	}
}
