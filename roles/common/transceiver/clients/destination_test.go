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

package clients

import (
	"testing"
	"time"

	"github.com/reinit/coward/roles/common/transceiver"
)

func TestDestination(t *testing.T) {
	reqs := requesters{
		req: make([]*requester, 3),
	}
	prios := make(priorities, 3)

	type example struct {
		reqDelay   time.Duration
		priosDelay time.Duration
	}

	examples := []example{
		example{
			reqDelay:   6 * time.Second,
			priosDelay: 6 * time.Second,
		},
		example{
			reqDelay:   1 * time.Second,
			priosDelay: 5 * time.Second,
		},
		example{
			reqDelay:   6 * time.Second,
			priosDelay: 1 * time.Second,
		},
	}

	for rIdx := range examples {
		reqs.req[rIdx] = &requester{
			id:        transceiver.ClientID(rIdx),
			requester: nil,
			sink:      false,
			delay:     dummyTimer{Delay: examples[rIdx].reqDelay},
		}

		prios[rIdx] = &priority{
			requester: reqs.req[rIdx],
			delay:     dummyTimer{Delay: examples[rIdx].priosDelay},
		}
	}

	d := destination{
		Priorities:   prios,
		ExpirerIndex: 0,
	}

	d.Renew()

	expected := []transceiver.ClientID{1, 2, 0}
	result := make([]transceiver.ClientID, 3)

	for dIdx := range d.Priorities {
		result[dIdx] = d.Priorities[dIdx].requester.ID()
	}

	for rIdx := range result {
		if result[rIdx] == expected[rIdx] {
			continue
		}

		t.Errorf("Expecting the item %d will be the client %d, got %d",
			rIdx, expected[rIdx], result[rIdx])

		return
	}
}
