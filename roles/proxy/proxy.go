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

package proxy

import (
	"time"

	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/network/server"
	"github.com/reinit/coward/roles/common/transceiver"
	tserver "github.com/reinit/coward/roles/common/transceiver/server"
	"github.com/reinit/coward/roles/proxy/common"
)

const (
	timeoutCheckTick = 300 * time.Millisecond
)

type proxy struct {
	listener        network.Listener
	cfg             Config
	logger          logger.Logger
	codec           transceiver.CodecBuilder
	mapping         common.Mapping
	serving         network.Serving
	runner          worker.Runner
	timeoutTicker   *time.Ticker
	unspawnNotifier role.UnspawnNotifier
}

// New creates a new proxy
func New(
	codec transceiver.CodecBuilder,
	l network.Listener,
	log logger.Logger,
	cfg Config,
) role.Role {
	proxyLog := log.Context("Proxy")

	return &proxy{
		listener:        l,
		cfg:             cfg,
		logger:          proxyLog,
		codec:           codec,
		mapping:         common.Mapping{},
		serving:         nil,
		runner:          nil,
		timeoutTicker:   nil,
		unspawnNotifier: nil,
	}
}

func (s *proxy) Spawn(unspawnNotifier role.UnspawnNotifier) error {
	s.unspawnNotifier = unspawnNotifier

	// Parse settings
	for mapIdx := range s.cfg.Mapping {
		s.mapping[s.cfg.Mapping[mapIdx].ID] = &common.Mapped{
			Protocol: s.cfg.Mapping[mapIdx].Protocol,
			Host:     s.cfg.Mapping[mapIdx].Host,
			Port:     s.cfg.Mapping[mapIdx].Port,
		}
	}

	// Start Corunner
	runner, runnerServeErr := worker.New(s.logger, worker.Config{
		MaxWorkers: (s.cfg.Capacity * uint32(
			s.cfg.ConnectionChannels)) + s.cfg.Capacity,
		MinWorkers: common.AutomaticalMinWorkerCount(
			(s.cfg.Capacity*uint32(s.cfg.ConnectionChannels))+s.cfg.Capacity,
			128),
		MaxWorkerIdle:     s.cfg.IdleTimeout * 10,
		JobReceiveTimeout: s.cfg.InitialTimeout,
	}).Serve()

	if runnerServeErr != nil {
		return runnerServeErr
	}

	s.runner = runner

	s.timeoutTicker = time.NewTicker(timeoutCheckTick)

	server, serveErr := server.New(s.listener, handler{
		transceiver: tserver.New(
			s.codec,
			s.timeoutTicker.C,
			tserver.Config{
				InitialTimeout:       s.cfg.InitialTimeout,
				IdleTimeout:          s.cfg.IdleTimeout,
				ConnectionChannels:   s.cfg.ConnectionChannels,
				ChannelDispatchDelay: s.cfg.ChannelDispatchDelay,
			},
		),
		runner:  s.runner,
		mapping: s.mapping,
		cfg:     s.cfg,
	}, s.logger, s.runner, server.Config{
		AcceptErrorWait: 300 * time.Millisecond,
		MaxConnections:  s.cfg.Capacity,
	}).Serve()

	if serveErr != nil {
		s.logger.Errorf("Failed to start server due to error: %s", serveErr)

		return serveErr
	}

	s.serving = server

	s.logger.Infof("Server is up, listening \"%s\"", s.serving.Listening())

	return nil
}

func (s *proxy) Unspawn() error {
	s.logger.Infof("Closing")

	if s.timeoutTicker != nil {
		s.timeoutTicker.Stop()
		s.timeoutTicker = nil
	}

	if s.serving != nil {
		closeErr := s.serving.Close()

		if closeErr != nil {
			s.logger.Errorf("Failed to close server due to error: %s", closeErr)

			return closeErr
		}

		s.serving = nil
	}

	if s.runner != nil {
		runnerCloseErr := s.runner.Close()

		if runnerCloseErr != nil {
			s.logger.Errorf(
				"Failed to close runner due to error: %s", runnerCloseErr)

			return runnerCloseErr
		}

		s.runner = nil
	}

	s.logger.Infof("Server is closed")

	s.unspawnNotifier <- struct{}{}

	return nil
}
