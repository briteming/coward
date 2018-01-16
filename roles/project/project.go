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
	"math"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	udpconn "github.com/reinit/coward/roles/common/network/connection/udp"
	"github.com/reinit/coward/roles/common/network/dialer/tcp"
	"github.com/reinit/coward/roles/common/network/dialer/udp"
	"github.com/reinit/coward/roles/common/transceiver"
	tclient "github.com/reinit/coward/roles/common/transceiver/client"
	"github.com/reinit/coward/roles/project/project"
	pcommon "github.com/reinit/coward/roles/proxy/common"
)

// Errors
var (
	ErrNoEndpointToProject = errors.New(
		"No endpoint to project")

	ErrUnknownEndpintNetworkProtocol = errors.New(
		"Unknown Endpoint protocol")
)

const (
	periodTickerDelay = 5 * time.Second
)

type projectile struct {
	codec           transceiver.CodecBuilder
	dialer          network.Dialer
	logger          logger.Logger
	cfg             Config
	transceiver     transceiver.Requester
	runner          worker.Runner
	projects        project.Projects
	periodTicker    *time.Ticker
	unspawnNotifier role.UnspawnNotifier
}

// New creates a new projectile
func New(
	codec transceiver.CodecBuilder,
	dialer network.Dialer,
	log logger.Logger,
	cfg Config,
) role.Role {
	return &projectile{
		codec:           codec,
		dialer:          dialer,
		logger:          log.Context("Project"),
		cfg:             cfg,
		transceiver:     nil,
		runner:          nil,
		periodTicker:    nil,
		unspawnNotifier: nil,
	}
}

// Spawn initialize a new Projectile
func (s *projectile) Spawn(unspawnNotifier role.UnspawnNotifier) error {
	s.unspawnNotifier = unspawnNotifier

	if len(s.cfg.Endpoints) <= 0 {
		return ErrNoEndpointToProject
	}

	// Start ticker
	s.periodTicker = time.NewTicker(periodTickerDelay)

	// Start Corunner
	runner, runnerServeErr := worker.New(s.logger, worker.Config{
		MaxWorkers: s.cfg.Endpoints.TotalConnections(),
		MinWorkers: pcommon.AutomaticalMinWorkerCount(
			s.cfg.Endpoints.TotalConnections(), 128),
		MaxWorkerIdle:     s.cfg.TransceiverIdleTimeout * 10,
		JobReceiveTimeout: s.cfg.TransceiverInitialTimeout,
	}).Serve()

	if runnerServeErr != nil {
		return runnerServeErr
	}

	s.runner = runner

	// Open transceiver client first
	trConnections := uint32(math.Ceil(float64(
		s.cfg.Endpoints.TotalConnections()) / float64(
		s.cfg.TransceiverChannels)))

	// Create a transceiver client without internal read timeout check ticker
	// so we only effected by the network failure rather than the internal
	// read timeout failure
	trServe, trServeErr := tclient.New(
		0, s.logger, s.dialer, s.codec, nil, tclient.Config{
			MaxConcurrent:        trConnections,
			RequestRetries:       1, // We'll do retry manually
			IdleTimeout:          s.cfg.TransceiverIdleTimeout,
			InitialTimeout:       s.cfg.TransceiverInitialTimeout,
			ConnectionPersistent: true, // Always use persistent connection
			ConnectionChannels:   s.cfg.TransceiverChannels,
		}).Serve()

	if trServeErr != nil {
		return trServeErr
	}

	s.transceiver = trServe

	pRegisterations := make([]project.Registeration, len(s.cfg.Endpoints))

	for epIdx := range s.cfg.Endpoints {
		switch s.cfg.Endpoints[epIdx].Protocol {
		case network.TCP:
			pRegisterations[epIdx] = project.Registeration{
				Endpoint: s.cfg.Endpoints[epIdx],
				Dialer: tcp.New(
					s.cfg.Endpoints[epIdx].Host, s.cfg.Endpoints[epIdx].Port,
					s.cfg.Endpoints[epIdx].RequestTimeout, tcpconn.Wrap),
				MinWorkers: pcommon.AutomaticalMinWorkerCount(
					s.cfg.Endpoints[epIdx].MaxConnections, 64),
			}

		case network.UDP:
			pRegisterations[epIdx] = project.Registeration{
				Endpoint: s.cfg.Endpoints[epIdx],
				Dialer: udp.New(
					s.cfg.Endpoints[epIdx].Host, s.cfg.Endpoints[epIdx].Port,
					s.cfg.Endpoints[epIdx].RequestTimeout, udpconn.Wrap),
				MinWorkers: pcommon.AutomaticalMinWorkerCount(
					s.cfg.Endpoints[epIdx].MaxConnections, 64),
			}

		default:
			return ErrUnknownEndpintNetworkProtocol
		}
	}

	pProjects, pProjectErr := project.New(
		s.logger,
		s.transceiver,
		s.runner,
		s.periodTicker.C,
		pRegisterations,
		project.Config{
			MaxConnections: trConnections,
			PingTickDelay:  s.cfg.TransceiverIdleTimeout / 2,
		})

	if pProjectErr != nil {
		return pProjectErr
	}

	s.projects = pProjects

	pBootErr := s.projects.Bootup()

	if pBootErr != nil {
		return pBootErr
	}

	s.logger.Infof("Ready")

	return nil
}

// Unspawn shuts down the Projectile
func (s *projectile) Unspawn() error {
	s.logger.Infof("Closing")

	if s.periodTicker != nil {
		s.periodTicker.Stop()
		s.periodTicker = nil
	}

	if s.projects != nil {
		// Kick first so no new transceiver connection can be created
		s.projects.Kick()
	}

	if s.transceiver != nil {
		cErr := s.transceiver.Close()

		if cErr != nil {
			s.logger.Errorf("Failed shutdown Transceiver: %s", cErr)

			return cErr
		}

		s.transceiver = nil
	}

	if s.projects != nil {
		s.projects.Close()
		s.projects = nil
	}

	if s.runner != nil {
		cErr := s.runner.Close()

		if cErr != nil {
			s.logger.Errorf("Failed shutdown Runner: %s", cErr)

			return cErr
		}

		s.runner = nil
	}

	s.unspawnNotifier <- struct{}{}

	s.logger.Infof("Server is down")

	return nil
}
