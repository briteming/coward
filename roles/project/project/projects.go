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
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/ticker"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/projector/projection"
)

// Errors
var (
	ErrRegisterAlreadyServing = errors.New(
		"Can't register new Projects during serving")
)

// Projects registeration and control
type Projects interface {
	Bootup() error
	Kick()
	Close()
}

// Endpoint endpoint data
type Endpoint struct {
	ID             projection.ID
	Host           string
	Port           uint16
	Protocol       network.Protocol
	MaxConnections uint32
	Retry          uint8
	Timeout        time.Duration
	RequestTimeout time.Duration
}

// Registeration Project registeration
type Registeration struct {
	Endpoint   Endpoint
	Dialer     network.Dialer
	MinWorkers uint32
}

// registered Projection
type registered struct {
	Project project
	Serving bool
}

// projects implements Projects
type projects struct {
	logger      logger.Logger
	projects    []registered
	connections *connections
	bootLock    sync.Mutex
	closeSignal chan struct{}
	closeWait   sync.WaitGroup
	serving     bool
}

// New creates a new Projects
func New(
	log logger.Logger,
	tClient transceiver.Requester,
	runner worker.Runner,
	ticker ticker.Requester,
	registerations []Registeration,
	cfg Config,
) (Projects, error) {
	prjs := &projects{
		logger:   log.Context("Project"),
		projects: make([]registered, len(registerations)),
		connections: &connections{
			connections: make(map[transceiver.ConnectionID]connection,
				cfg.MaxConnections),
		},
		bootLock:    sync.Mutex{},
		closeSignal: make(chan struct{}, 1),
		closeWait:   sync.WaitGroup{},
		serving:     false,
	}

	projsIdx := 0

	for rrIdx := range registerations {
		prjs.projects[projsIdx] = registered{
			Project: project{
				endpoint:           registerations[rrIdx].Endpoint,
				dialer:             registerations[rrIdx].Dialer,
				minWorkers:         registerations[rrIdx].MinWorkers,
				pingTickTimeout:    cfg.PingTickDelay,
				connRequestTimeout: cfg.RequestTimeout,
				runner:             runner,
				connections:        prjs.connections,
				lastWorkID:         0,
				startedWorkers:     0,
				idleWorkers:        0,
				logger: prjs.logger.Context("Projection " +
					"(#" + strconv.FormatUint(
					uint64(registerations[rrIdx].Endpoint.ID), 10) + " " +
					strconv.FormatUint(uint64(rrIdx), 10) +
					") " +
					registerations[rrIdx].Endpoint.Protocol.String() +
					" " + net.JoinHostPort(
					registerations[rrIdx].Endpoint.Host,
					strconv.FormatUint(uint64(
						registerations[rrIdx].Endpoint.Port), 10))),
				transceiver: tClient,
				ticker:      ticker,
				closeSignal: prjs.closeSignal,
				closeWait:   &prjs.closeWait,
				closing:     false,
				lock:        &prjs.bootLock,
			},
			Serving: false,
		}

		projsIdx++
	}

	return prjs, nil
}

// Bootup start serving projects
func (p *projects) Bootup() error {
	p.bootLock.Lock()
	defer p.bootLock.Unlock()

	for pIdx := range p.projects {
		p.projects[pIdx].Project.Initialize()
	}

	return nil
}

// Kick notify all projects to get to closed.
func (p *projects) Kick() {
	p.bootLock.Lock()
	defer p.bootLock.Unlock()

	for pIdx := range p.projects {
		p.projects[pIdx].Project.Kick()
	}

	close(p.closeSignal)
}

// Close wait projects to shutdown and clear resource
func (p *projects) Close() {
	p.closeWait.Wait()

	p.bootLock.Lock()
	defer p.bootLock.Unlock()

	p.connections.Clear()
}
