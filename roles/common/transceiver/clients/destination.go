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
	"sort"
	"time"

	"github.com/reinit/coward/common/timer"
)

type priority struct {
	requester *requester
	delay     timer.Timer
}

type priorities []*priority

type destination struct {
	Priorities      priorities
	ExpirerIndex    int
	RunningRequests uint32
}

func (p *priority) Delay() timer.Timer {
	return p.delay
}

func (p *priority) Duration() time.Duration {
	return p.requester.Delay().Duration() + p.delay.Duration()
}

func (d priorities) Len() int {
	return len(d)
}

func (d priorities) Less(i, j int) bool {
	if d[i].requester.sink && !d[j].requester.sink {
		return false
	}

	return d[i].Duration() < d[j].Duration()
}

func (d priorities) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d *destination) Renew() {
	if !d.Outdated() {
		return
	}

	sort.Sort(d.Priorities)
}

func (d *destination) Outdated() bool {
	if d.Priorities.Len() < 2 {
		return false
	}

	if d.Priorities.Less(0, 1) {
		return false
	}

	return (d.Priorities[0].Duration() / 2) > d.Priorities[1].Duration()
}
