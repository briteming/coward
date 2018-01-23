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

package socks5

import (
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/common/ticker"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/network/server"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/common/transceiver/clients"
	pcommon "github.com/reinit/coward/roles/proxy/common"
	"github.com/reinit/coward/roles/socks5/common"
)

// Authenticator is the Socks5 User Authenticator function
type Authenticator func(username, password string) error

type socks5 struct {
	clients         transceiver.Balancer
	listener        network.Listener
	log             logger.Logger
	cfg             Config
	transceiver     transceiver.Balanced
	ticker          ticker.RequestCloser
	serverServing   network.Serving
	runner          worker.Runner
	unspawnNotifier role.UnspawnNotifier
}

// New creates a new Socks5 server
func New(
	ticker ticker.RequestCloser,
	cs []transceiver.Client,
	listener network.Listener,
	log logger.Logger,
	cfg Config,
) role.Role {
	return &socks5{
		clients:         clients.New(cs, cfg.MaxDestinationRecords),
		listener:        listener,
		log:             log.Context("Socks5"),
		cfg:             cfg,
		transceiver:     nil,
		ticker:          ticker,
		serverServing:   nil,
		runner:          nil,
		unspawnNotifier: nil,
	}
}

func (s *socks5) Spawn(unspawnNotifier role.UnspawnNotifier) error {
	s.unspawnNotifier = unspawnNotifier

	// Open transceiver client first
	trServes, trServeErr := s.clients.Serve()

	if trServeErr != nil {
		s.log.Errorf("Failed to start Transceivers due to error: %s",
			trServeErr)

		return trServeErr
	}

	s.transceiver = trServes

	// Start Corunner
	runner, runnerServeErr := worker.New(s.log, worker.Config{
		MaxWorkers: s.cfg.Capacity * 2,
		MinWorkers: pcommon.AutomaticalMinWorkerCount(
			s.cfg.Capacity*2, 128),
		MaxWorkerIdle:     s.cfg.ConnectionTimeout * 10,
		JobReceiveTimeout: s.cfg.NegotiationTimeout,
	}).Serve()

	if runnerServeErr != nil {
		return runnerServeErr
	}

	s.runner = runner

	// Build Transceiver Client read buffer
	shb := &common.SharedBuffers{
		Buf: make([]*common.SharedBuffer, s.transceiver.Size()),
	}

	s.transceiver.Clients(func(
		client transceiver.ClientID,
		req transceiver.Requester,
	) {
		shb.Buf[client] = &common.SharedBuffer{
			Buffer: make([]byte, 4096*req.Connections()),
			Size:   4096,
		}
	})

	// Then, start server
	serverServing, serverServeErr := server.New(s.listener, handler{
		runner:        s.runner,
		shb:           shb,
		transceiver:   s.transceiver,
		negoTimeout:   s.cfg.NegotiationTimeout,
		timeout:       s.cfg.ConnectionTimeout,
		authenticator: s.cfg.Authenticator,
	}, s.log, s.runner, server.Config{
		AcceptErrorWait: 100 * time.Millisecond,
		MaxConnections:  s.cfg.Capacity,
	}).Serve()

	if serverServeErr != nil {
		s.log.Errorf("Failed to start server due to error: %s", serverServeErr)

		return serverServeErr
	}

	s.serverServing = serverServing

	s.log.Infof("Server is up, listening \"%s\"", s.serverServing.Listening())

	return nil
}

func (s *socks5) Unspawn() error {
	s.log.Infof("Closing")

	// It seems a bit counterintuitive, but we had to shutdown transceiver
	// first to prevent ongoing requests block the server from shutting down
	// (consider when requests running on a dead transceiver connection waiting
	// for relay to confirm it's close signal)
	if s.transceiver != nil {
		trsmCloseErr := s.transceiver.Close()

		if trsmCloseErr != nil {
			s.log.Errorf(
				"Failed to close Transceiver due to error: %s", trsmCloseErr)

			return trsmCloseErr
		}

		s.transceiver = nil
	}

	if s.serverServing != nil {
		serverCloseErr := s.serverServing.Close()

		if serverCloseErr != nil {
			s.log.Errorf("Failed to close server due to error: %s",
				serverCloseErr)

			return serverCloseErr
		}

		s.serverServing = nil
	}

	if s.ticker != nil {
		s.ticker.Close()
		s.ticker = nil
	}

	if s.runner != nil {
		runnerCloseErr := s.runner.Close()

		if runnerCloseErr != nil {
			s.log.Errorf("Failed to close runner due to error: %s",
				runnerCloseErr)

			return runnerCloseErr
		}

		s.runner = nil
	}

	s.log.Infof("Server is closed")

	s.unspawnNotifier <- struct{}{}

	return nil
}
