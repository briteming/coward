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
	"sort"
	"time"

	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/roles/common/transceiver"
)

type requester struct {
	id        transceiver.ClientID
	requester transceiver.Requester
	sink      bool
	delay     timer.Timer
}

type requesters struct {
	req        []*requester
	lastUpdate time.Time
}

func (r *requester) Sink(s bool) {
	r.sink = s
}

func (r *requester) ID() transceiver.ClientID {
	return r.id
}

func (r *requester) Delay() timer.Timer {
	return r.delay
}

func (r *requester) Request(
	req transceiver.RequestBuilder,
	cancel <-chan struct{},
	m transceiver.Meter,
) (bool, error) {
	return r.requester.Request(req, cancel, m)
}

func (r *requesters) Renew() {
	if !r.Outdated() {
		return
	}

	sort.Sort(r)

	r.lastUpdate = time.Now()
}

func (r *requesters) Updated() time.Time {
	return r.lastUpdate
}

func (r *requesters) Outdated() bool {
	if r.Len() < 2 {
		return false
	}

	if r.Less(0, 1) {
		return false
	}

	return true
}

func (r *requesters) All(iter func(idx int, req *requester)) {
	for v := range r.req {
		iter(v, r.req[v])
	}
}

func (r *requesters) Len() int {
	return len(r.req)
}

func (r *requesters) Less(i, j int) bool {
	if r.req[i].sink && !r.req[j].sink {
		return false
	}

	return r.req[i].delay.Duration() < r.req[j].delay.Duration()
}

func (r *requesters) Swap(i, j int) {
	r.req[i], r.req[j] = r.req[j], r.req[i]
}
