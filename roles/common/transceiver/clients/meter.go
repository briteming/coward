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
	"sync"
	"sync/atomic"
	"time"

	"github.com/reinit/coward/common/timer"
)

type meter struct {
	current     *priority
	requesters  *requesters
	destination *destination
	lock        *sync.Mutex
}

type meterConnectionStopper struct {
	requesters *requesters
	stopper    timer.Stopper
	lock       *sync.Mutex
}

type meterRequestStopper struct {
	dest    *destination
	stopper timer.Stopper
	lock    *sync.Mutex
}

func (m meterConnectionStopper) Stop() time.Duration {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Sync the value into requester before doing anything else
	duration := m.stopper.Stop()

	m.requesters.Renew()

	return duration
}

func (m meterRequestStopper) Stop() time.Duration {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Sync the value into destination record before doing anything else
	duration := m.stopper.Stop()

	// Don't resort if somebody is requesting
	if atomic.LoadUint32(&m.dest.RunningRequests) > 1 {
		return duration
	}

	m.dest.Renew()

	return duration
}

func (m *meter) Connection() timer.Stopper {
	return meterConnectionStopper{
		requesters: m.requesters,
		stopper:    m.current.requester.Delay().Start(),
		lock:       m.lock,
	}
}

func (m *meter) ConnectionFailure(e error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	switch e {
	case nil:
		m.current.requester.Sink(false)

	default:
		m.current.requester.Sink(true)
	}
}

func (m *meter) Request() timer.Stopper {
	return meterRequestStopper{
		dest:    m.destination,
		stopper: m.current.Delay().Start(),
		lock:    m.lock,
	}
}
