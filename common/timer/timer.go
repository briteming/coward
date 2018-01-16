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

// Timer is the timer
type Timer interface {
	Duration() time.Duration
	Start() Stopper
	Reset()
}

// Stopper is the timer stopper
type Stopper interface {
	Stop() time.Duration
}

// timer implements Timer
type timer struct {
	duration time.Duration
}

// stopper implements Stopper
type stopper struct {
	timer *timer
	start time.Time
}

// New creates a new Timer
func New() Timer {
	return &timer{
		duration: 0,
	}
}

// Duration returns the recorded time duration
func (t timer) Duration() time.Duration {
	return t.duration
}

// Start starts the timer
func (t *timer) Start() Stopper {
	return stopper{
		timer: t,
		start: time.Now(),
	}
}

// Reset resets the timer
func (t *timer) Reset() {
	t.duration = 0
}

// Stop stops the timer and collect the result
func (t stopper) Stop() time.Duration {
	t.timer.duration = time.Now().Sub(t.start)

	return t.timer.duration
}
