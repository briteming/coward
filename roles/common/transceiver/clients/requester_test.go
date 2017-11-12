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
	"strconv"
	"testing"
	"time"

	"github.com/reinit/coward/common/timer"
)

type dummyTimer struct {
	Delay time.Duration
}

func (d dummyTimer) Duration() time.Duration {
	return d.Delay
}

func (d dummyTimer) Start() timer.Stopper {
	return nil
}

func (d dummyTimer) Reset() {}

func (d dummyTimer) Stop() time.Duration {
	return d.Delay
}

func TestRequester(t *testing.T) {
	requester1 := &requester{
		id:        0,
		requester: nil,
		sink:      false,
		delay:     dummyTimer{Delay: 10 * time.Second},
	}
	requester2 := &requester{
		id:        1,
		requester: nil,
		sink:      false,
		delay:     dummyTimer{Delay: 0},
	}
	requester3 := &requester{
		id:        2,
		requester: nil,
		sink:      true,
		delay:     dummyTimer{Delay: 5 * time.Second},
	}

	reqs := requesters{
		req:        []*requester{requester1, requester2, requester3},
		lastUpdate: time.Time{},
	}

	reqs.Renew()

	result := " < "

	reqs.All(func(idx int, req *requester) {
		result += strconv.FormatUint(uint64(req.ID()), 10) + " < "
	})

	if result != " < 1 < 0 < 2 < " {
		t.Errorf("Failed to sort the request into expected order. "+
			"Expecting %s, got %s", "< 1 < 0 < 2 <", result)

		return
	}
}
