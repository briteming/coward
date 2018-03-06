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

package projector

import (
	"errors"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/common/ticker"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	udpconn "github.com/reinit/coward/roles/common/network/connection/udp"
	tcplistener "github.com/reinit/coward/roles/common/network/listener/tcp"
	udplistener "github.com/reinit/coward/roles/common/network/listener/udp"
	"github.com/reinit/coward/roles/common/network/server"
	"github.com/reinit/coward/roles/common/transceiver"
	tserver "github.com/reinit/coward/roles/common/transceiver/server"
	"github.com/reinit/coward/roles/projector/projection"
	"github.com/reinit/coward/roles/proxy/common"
)

// Errors
var (
	ErrNoServerToProject = errors.New(
		"No server is defined")

	ErrUnsupportedNetworkProtocolType = errors.New(
		"Unsupported network protocol type")
)

// Consts
const (
	tickDelay = 300 * time.Millisecond
)

// projector Projector
type projector struct {
	logger          logger.Logger
	codec           transceiver.CodecBuilder
	cfg             Config
	listener        network.Listener
	unspawnNotifier role.UnspawnNotifier
	runner          worker.Runner
	ticker          ticker.RequestCloser
	tserver         network.Serving
	servers         []network.Serving
	projections     projection.Projections
}

// New creates a new Projector
func New(
	listener network.Listener,
	codec transceiver.CodecBuilder,
	log logger.Logger,
	cfg Config,
) role.Role {
	return &projector{
		logger:          log.Context("Projector"),
		codec:           codec,
		cfg:             cfg,
		listener:        listener,
		unspawnNotifier: nil,
		runner:          nil,
		ticker:          nil,
		tserver:         nil,
		servers:         []network.Serving{},
		projections:     nil,
	}
}

// Spawn initialize current Projector
func (s *projector) Spawn(unspawnNotifier role.UnspawnNotifier) error {
	s.unspawnNotifier = unspawnNotifier

	if len(s.cfg.Servers) <= 0 {
		return ErrNoServerToProject
	}

	tticker, tickerErr := ticker.New(tickDelay, 1024).Serve()

	if tickerErr != nil {
		return tickerErr
	}

	s.ticker = tticker

	// Start Corunner
	startWorkers := s.cfg.GetTotalServerCapacity() +
		((s.cfg.Capacity * uint32(s.cfg.ConnectionChannels)) * 2)

	runner, runnerServeErr := worker.New(s.logger, s.ticker, worker.Config{
		MaxWorkers:        startWorkers,
		MinWorkers:        common.AutomaticalMinWorkerCount(startWorkers, 128),
		MaxWorkerIdle:     s.cfg.IdleTimeout * 10,
		JobReceiveTimeout: s.cfg.InitialTimeout,
	}).Serve()

	if runnerServeErr != nil {
		return runnerServeErr
	}

	s.runner = runner

	// Initialize Project manager
	s.projections = projection.New(
		s.runner, s.ticker, projection.Config{
			MaxReceivers:   s.cfg.Capacity,
			RequestTimeout: s.cfg.InitialTimeout,
			Projects:       s.cfg.GetAllServerRegisterations(),
		})

	// Bootup project servers
	s.servers = make([]network.Serving, len(s.cfg.Servers))

	serverIdx := 0

	for sIdx := range s.cfg.Servers {
		var serving network.Serving
		var serveErr error

		pHandler, pErr := s.projections.Handler(s.cfg.Servers[sIdx].ID)

		if pErr != nil {
			return pErr
		}

		switch s.cfg.Servers[sIdx].Protocol {
		case network.TCP:
			serving, serveErr = server.New(tcplistener.New(
				s.cfg.Servers[sIdx].Interface,
				s.cfg.Servers[sIdx].Port,
				tcpconn.Wrap,
			), pHandler, s.logger.Context("Projection (#"+
				strconv.FormatUint(uint64(s.cfg.Servers[sIdx].ID), 10)+" "+
				strconv.FormatInt(int64(sIdx), 10)+
				") TCP "+net.JoinHostPort(
				s.cfg.Servers[sIdx].Interface.String(),
				strconv.FormatUint(uint64(
					s.cfg.Servers[sIdx].Port), 10))), s.runner, server.Config{
				MaxConnections:  s.cfg.Servers[sIdx].Capacity,
				AcceptErrorWait: 300 * time.Millisecond,
			}).Serve()

		case network.UDP:
			serving, serveErr = server.New(udplistener.New(
				s.cfg.Servers[sIdx].Interface,
				s.cfg.Servers[sIdx].Port,
				s.cfg.Servers[sIdx].RequestTimeout,
				s.cfg.Servers[sIdx].Capacity,
				make([]byte, 4096),
				s.ticker,
				udpconn.Wrap,
			), pHandler, s.logger.Context("Projection (#"+
				strconv.FormatUint(uint64(s.cfg.Servers[sIdx].ID), 10)+" "+
				strconv.FormatInt(int64(sIdx), 10)+
				") UDP "+net.JoinHostPort(
				s.cfg.Servers[sIdx].Interface.String(),
				strconv.FormatUint(uint64(
					s.cfg.Servers[sIdx].Port), 10))), s.runner, server.Config{
				MaxConnections:  s.cfg.Servers[sIdx].Capacity,
				AcceptErrorWait: 300 * time.Millisecond,
			}).Serve()

		default:
			return ErrUnsupportedNetworkProtocolType
		}

		if serveErr != nil {
			s.logger.Errorf("Failed to boot up server for %s Projection %d "+
				"on \"%s\" due to error: %s",
				s.cfg.Servers[sIdx].Protocol,
				s.cfg.Servers[sIdx].ID, net.JoinHostPort(
					s.cfg.Servers[sIdx].Interface.String(),
					strconv.FormatUint(uint64(
						s.cfg.Servers[sIdx].Port), 10)), serveErr)

			return serveErr
		}

		s.logger.Infof("Serving %s Projection %d on \"%s\"",
			s.cfg.Servers[sIdx].Protocol,
			s.cfg.Servers[sIdx].ID, serving.Listening())

		s.servers[serverIdx] = serving

		serverIdx++
	}

	minTimeout := s.cfg.IdleTimeout / time.Second

	if minTimeout > math.MaxUint16 {
		minTimeout = math.MaxUint16
	}

	server, serveErr := server.New(s.listener, handler{
		transceiver: tserver.New(s.codec, nil, tserver.Config{
			InitialTimeout:       s.cfg.InitialTimeout,
			IdleTimeout:          s.cfg.IdleTimeout,
			ConnectionChannels:   s.cfg.ConnectionChannels,
			ChannelDispatchDelay: s.cfg.ChannelDispatchDelay,
		}),
		runner:      s.runner,
		projections: s.projections,
		minTimeout:  uint16(minTimeout),
		cfg:         s.cfg,
	}, s.logger, s.runner, server.Config{
		MaxConnections:  s.cfg.Capacity,
		AcceptErrorWait: 300 * time.Millisecond,
	}).Serve()

	if serveErr != nil {
		s.logger.Errorf("Failed to start server due to error: %s", serveErr)

		return serveErr
	}

	s.tserver = server

	s.logger.Infof("Serving, register at \"%s\"", s.tserver.Listening())

	return nil
}

// // Unspawn closes current Projector
func (s *projector) Unspawn() error {
	s.logger.Infof("Closing")

	for sIdx := range s.servers {
		if s.servers[sIdx] == nil {
			continue
		}

		closeErr := s.servers[sIdx].Close()

		if closeErr != nil {
			s.logger.Warningf("Failed to shutdown projection %d (%s)"+
				" due to error: %s", sIdx, s.servers[sIdx].Listening(),
				closeErr)
		}

		s.logger.Infof("Projection %d (%s) has been closed",
			sIdx, s.servers[sIdx].Listening())

		s.servers[sIdx] = nil
	}

	if s.tserver != nil {
		closeErr := s.tserver.Close()

		if closeErr != nil {
			s.logger.Errorf("Failed to shutdown Transceivers due to error: %s",
				closeErr)
		}

		s.tserver = nil
	}

	if s.runner != nil {
		closeErr := s.runner.Close()

		if closeErr != nil {
			s.logger.Errorf("Failed to shutdown runner due to error: %s",
				closeErr)
		}

		s.runner = nil
	}

	if s.ticker != nil {
		s.ticker.Close()
		s.ticker = nil
	}

	s.logger.Infof("Server is closed")

	s.unspawnNotifier <- struct{}{}

	return nil
}
