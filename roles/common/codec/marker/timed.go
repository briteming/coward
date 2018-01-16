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

package marker

import (
	"errors"
	"time"
)

// Errors
var (
	ErrAlreadyExisted = errors.New(
		"Marker already existed")
)

type timed struct {
	markers     []map[Mark]struct{}
	timer       func() time.Time
	nextSwitch  time.Time
	switchDelay time.Duration
	index       uint8
	cap         int
}

// Timed creates a timed (expirable) Marker
func Timed(capSize int, d time.Duration, t func() time.Time) Marker {
	return &timed{
		markers:     make([]map[Mark]struct{}, 3),
		timer:       t,
		nextSwitch:  t().Add(d),
		switchDelay: d,
		index:       0,
		cap:         0,
	}
}

func (t *timed) nextIndex(currentIndex uint8) uint8 {
	return (currentIndex + 1) % 3
}

func (t *timed) currentIndex() uint8 {
	if t.timer().After(t.nextSwitch) {
		t.markers[t.index] = nil

		t.index = t.nextIndex(t.index)
		t.nextSwitch = time.Now().Add(t.switchDelay)
	}

	return t.index
}

func (t *timed) getMap(index uint8) map[Mark]struct{} {
	if t.markers[index] == nil {
		t.markers[index] = make(map[Mark]struct{}, t.cap)
	}

	return t.markers[index]
}

func (t *timed) Mark(m Mark) error {
	cIdx := t.currentIndex()

	_, found := t.getMap(cIdx)[m]

	if found {
		return ErrAlreadyExisted
	}

	t.getMap(cIdx)[m] = struct{}{}
	t.getMap(t.nextIndex(cIdx))[m] = struct{}{}

	return nil
}
