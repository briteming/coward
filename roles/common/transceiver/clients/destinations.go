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
	"errors"
	"sync"
	"sync/atomic"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/roles/common/transceiver"
)

// Errors
var (
	ErrDestinationsAllDestinationsHasFailed = errors.New(
		"No Transceiver Clients can complete the request")
)

type destinations struct {
	dest   map[transceiver.Destination]*destination
	expire expirer
}

func (d *destinations) getDest(
	name transceiver.Destination,
	requesters *requesters) *destination {
	de, deFound := d.dest[name]

	if deFound {
		bumped, expireID := d.expire.Bump(de.ExpirerIndex)

		de.ExpirerIndex = expireID

		if bumped != "" {
			delete(d.dest, bumped)
		}

		return de
	}

	bumped, expireID := d.expire.Add(name)

	de = &destination{
		Priorities:   make(priorities, requesters.Len()),
		ExpirerIndex: expireID,
	}

	requesters.All(func(idx int, req *requester) {
		de.Priorities[idx] = &priority{
			requester: req,
			delay:     timer.Average(),
		}
	})

	d.dest[name] = de

	delete(d.dest, bumped)

	return de
}

func (d *destinations) Request(
	log logger.Logger,
	dest transceiver.Destination,
	req transceiver.BalancedRequestBuilder,
	cancel <-chan struct{},
	requesters *requesters,
	lock *sync.Mutex,
) error {
	var retriable bool
	var reqErr error

	// Load destination record
	lock.Lock()
	dests := d.getDest(dest, requesters)
	destPriorities := make(priorities, dests.Priorities.Len())

	copy(destPriorities, dests.Priorities)
	lock.Unlock()

	continueLoop := true
	serverTried := make([]bool, len(destPriorities))

	for stIdx := range serverTried {
		serverTried[stIdx] = false
	}

	atomic.AddUint32(&dests.RunningRequests, 1)
	defer atomic.AddUint32(&dests.RunningRequests, ^uint32(0))

	// Try all Clients one by one
	for {
		for dIdx := range destPriorities {
			if serverTried[dIdx] {
				continue
			}

			if continueLoop &&
				!destPriorities[dIdx].requester.requester.Available() &&
				destPriorities[dIdx].requester.requester.Full() {
				continue
			}

			m := &meter{
				current:     destPriorities[dIdx],
				requesters:  requesters,
				destination: dests,
				lock:        lock,
			}

			serverTried[dIdx] = true

			retriable, reqErr = destPriorities[dIdx].requester.Request(
				log, func(
					connectionID transceiver.ConnectionID,
					server rw.ReadWriteDepleteDoner,
					connCtl transceiver.ConnectionControl,
					log logger.Logger,
				) fsm.Machine {
					return req(
						destPriorities[dIdx].requester.ID(),
						connectionID,
						server,
						connCtl,
						log)
				}, cancel, m)

			if reqErr == nil {
				return nil
			}

			if retriable {
				continue
			}

			return reqErr
		}

		if !continueLoop {
			break
		}

		continueLoop = false
	}

	if reqErr != nil {
		return reqErr
	}

	return ErrDestinationsAllDestinationsHasFailed
}
