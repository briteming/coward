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

package project

import (
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
)

type project struct {
	endpoint        Endpoint
	dialer          network.Dialer
	minWorkers      uint32
	pingTickTimeout time.Duration
	runner          worker.Runner
	lastWorkID      uint64
	startedWorkers  uint32
	idleWorkers     uint32
	connections     *connections
	logger          logger.Logger
	transceiver     transceiver.Requester
	closeSignal     chan struct{}
	closeWait       *sync.WaitGroup
	closing         bool
	lock            *sync.Mutex
}

func (p *project) persistRequest(
	cID transceiver.ConnectionID,
	server rw.ReadWriteDepleteDoner,
	log logger.Logger,
) fsm.Machine {
	p.lock.Lock()
	defer p.lock.Unlock()

	connData := p.connections.Get(cID)

	return &requester{
		projection:            p.endpoint,
		dialer:                p.dialer,
		runner:                p.runner,
		buf:                   connData.Buffer,
		project:               p,
		pingTickTimeout:       p.pingTickTimeout,
		idleQuitTimeout:       p.endpoint.Timeout,
		currentRelay:          nil,
		pendingRelay:          false,
		keepAliveResult:       nil,
		keepAlivePingTicker:   connData.PingTicker,
		keepAlivePeriodTicker: nil,
		keepAliveTickResume:   make(chan requesterKeepAliveTickResumer, 1),
		keepAliveQuit:         make(chan struct{}, 1),
		rw:                    server,
		log:                   log,
		serverPingDelay:       0,
	}
}

func (p *project) request(
	cID transceiver.ConnectionID,
	server rw.ReadWriteDepleteDoner,
	log logger.Logger,
) fsm.Machine {
	p.lock.Lock()
	defer p.lock.Unlock()

	connData := p.connections.Get(cID)

	return &requester{
		projection:            p.endpoint,
		dialer:                p.dialer,
		runner:                p.runner,
		buf:                   connData.Buffer,
		project:               p,
		pingTickTimeout:       p.pingTickTimeout,
		idleQuitTimeout:       p.endpoint.Timeout,
		currentRelay:          nil,
		pendingRelay:          false,
		keepAliveResult:       nil,
		keepAlivePingTicker:   connData.PingTicker,
		keepAlivePeriodTicker: connData.PeriodTicker,
		keepAliveTickResume:   make(chan requesterKeepAliveTickResumer, 1),
		keepAliveQuit:         make(chan struct{}, 1),
		rw:                    server,
		log:                   log,
		serverPingDelay:       0,
	}
}

func (p *project) handle(
	id uint64,
	reqBuilder transceiver.RequestBuilder,
	retry bool,
) {
	defer func() {
		p.lock.Lock()
		defer p.lock.Unlock()

		p.startedWorkers--

		p.closeWait.Done()
	}()

	ll := p.logger.Context("Request (" + strconv.FormatUint(id, 10) + ")")

	m := meter{
		connection: timer.New(),
		request:    timer.New(),
	}

	for {
		select {
		case <-p.closeSignal:
			return

		default:
			ll.Debugf("Serving")

			_, tReqErr := p.transceiver.Request(ll, reqBuilder, nil, m)

			if tReqErr == nil {
				if retry {
					ll.Debugf("Completed. Connection delay %s, request "+
						"delay %s. Restarting",
						m.connection.Duration(), m.request.Duration())

					continue
				}

				ll.Debugf("Completed. Connection delay %s, request delay %s",
					m.connection.Duration(), m.request.Duration())

				return
			}

			if retry {
				ll.Warningf("Can't serve due to error: %s. Restarting", tReqErr)

				retryAt := time.Now().Add(p.endpoint.RequestTimeout)

				for {
					time.Sleep(1 * time.Second)

					select {
					case <-p.closeSignal:
						return

					default:
					}

					if time.Now().After(retryAt) {
						break
					}
				}

				continue
			}

			ll.Warningf("Can't serve due to error: %s", tReqErr)

			return
		}
	}
}

func (p *project) startHandle(
	reqBuilder transceiver.RequestBuilder,
	retry bool,
) {
	if p.startedWorkers >= p.endpoint.MaxConnections {
		return
	}

	remainToStart := p.endpoint.MaxConnections - p.startedWorkers

	if remainToStart > p.minWorkers {
		remainToStart = p.minWorkers
	}

	for iStarted := uint32(0); iStarted < remainToStart; iStarted++ {
		p.startedWorkers++
		p.lastWorkID++

		if p.lastWorkID >= math.MaxUint64-1 {
			p.lastWorkID = 0
		}

		p.closeWait.Add(1)

		go p.handle(p.lastWorkID, reqBuilder, retry)
	}
}

func (p *project) replenish() {
	if p.idleWorkers >= p.minWorkers {
		return
	}

	p.startHandle(p.request, false)
}

func (p *project) IncreaseIdleCount() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.idleWorkers++
}

func (p *project) DecreaseIdleCount() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.idleWorkers--

	if p.closing {
		return
	}

	p.replenish()
}

func (p *project) Initialize() {
	p.lastWorkID = 0
	p.startedWorkers = 0
	p.idleWorkers = 0

	p.startHandle(p.persistRequest, true)
}

func (p *project) Kick() {
	p.closing = true
}
