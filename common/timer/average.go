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

package timer

import "time"

const (
	maxAvgSlots = 8
)

// average implements Timer
type average struct {
	slots       [maxAvgSlots]time.Duration
	avgFactor   time.Duration
	currentSlot int
	total       time.Duration
	avg         time.Duration
}

// averageStopper implements Stopper
type averageStopper struct {
	startTime time.Time
	average   *average
}

// Average creates a new average
func Average() Timer {
	return &average{
		slots:       [maxAvgSlots]time.Duration{},
		avgFactor:   0,
		currentSlot: 0,
		total:       0,
		avg:         0,
	}
}

// Duration returns the average time recorded by current timer
func (t average) Duration() time.Duration {
	return t.avg
}

// Start start current timer
func (t *average) Start() Stopper {
	return averageStopper{
		startTime: time.Now(),
		average:   t,
	}
}

// Reset resets the Average Timer
func (t *average) Reset() {
	t.total = 0
	t.avg = 0
	t.slots = [maxAvgSlots]time.Duration{}
	t.avgFactor = 0
}

// Stop stops current timer and returns the average time
func (s averageStopper) Stop() time.Duration {
	resultTime := time.Now().Sub(s.startTime)

	if maxAvgSlots > s.average.avgFactor {
		s.average.avgFactor++
	}

	s.average.total -= s.average.slots[s.average.currentSlot]

	s.average.slots[s.average.currentSlot] = resultTime
	s.average.total += resultTime

	s.average.avg = s.average.total / s.average.avgFactor

	s.average.currentSlot++
	s.average.currentSlot %= maxAvgSlots

	return s.average.avg
}
