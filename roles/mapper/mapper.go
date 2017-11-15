//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2017 Rui NI <ranqus@gmail.com>
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

package mapper

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/roles/common/network"
	tcpconn "github.com/reinit/coward/roles/common/network/connection/tcp"
	udpconn "github.com/reinit/coward/roles/common/network/connection/udp"
	tcplistener "github.com/reinit/coward/roles/common/network/listener/tcp"
	udplistener "github.com/reinit/coward/roles/common/network/listener/udp"
	"github.com/reinit/coward/roles/common/network/server"
	"github.com/reinit/coward/roles/common/transceiver"
	tclient "github.com/reinit/coward/roles/common/transceiver/client"
	"github.com/reinit/coward/roles/mapper/common"
	pcommon "github.com/reinit/coward/roles/proxy/common"
)

// Errors
var (
	ErrNoTargetForMapping = errors.New(
		"No target for mapping")

	ErrUnknownNetProtoType = errors.New(
		"Unknow network protocol type")
)

const (
	trReadTimeoutTickDelay = 300 * time.Millisecond
)

type mapper struct {
	codec               transceiver.CodecBuilder
	dialer              network.Dialer
	log                 logger.Logger
	cfg                 Config
	transceiver         transceiver.Requester
	trReadTimeoutTicker *time.Ticker
	servers             []network.Serving
	runner              worker.Runner
	unspawnNotifier     role.UnspawnNotifier
}

// New creates a new Mapper
func New(
	codec transceiver.CodecBuilder,
	dialer network.Dialer,
	log logger.Logger,
	cfg Config,
) role.Role {
	return &mapper{
		codec:               codec,
		dialer:              dialer,
		log:                 log.Context("Mapper"),
		cfg:                 cfg,
		transceiver:         nil,
		trReadTimeoutTicker: nil,
		servers:             nil,
		runner:              nil,
		unspawnNotifier:     nil,
	}
}

func (s *mapper) Spawn(unspawnNotifier role.UnspawnNotifier) error {
	s.unspawnNotifier = unspawnNotifier

	if len(s.cfg.Mapping) < 1 {
		return ErrNoTargetForMapping
	}

	s.trReadTimeoutTicker = time.NewTicker(trReadTimeoutTickDelay)

	// Open transceiver client first
	trServe, trServeErr := tclient.New(
		0, s.log, s.dialer, s.codec, s.trReadTimeoutTicker.C, tclient.Config{
			MaxConcurrent:        s.cfg.TransceiverMaxConnections,
			RequestRetries:       s.cfg.TransceiverRequestRetries,
			IdleTimeout:          s.cfg.TransceiverIdleTimeout,
			InitialTimeout:       s.cfg.TransceiverInitialTimeout,
			ConnectionPersistent: s.cfg.TransceiverConnectionPersistent,
			ConnectionChannels:   s.cfg.TransceiverChannels,
		}).Serve()

	if trServeErr != nil {
		s.log.Errorf("Failed to start Transceiver due to error: %s", trServeErr)

		return trServeErr
	}

	s.transceiver = trServe

	// Init runners
	runner, runnerServeErr := worker.New(s.log, worker.Config{
		MaxWorkers: trServe.Channels() + s.cfg.Mapping.TotalCapacity(),
		MinWorkers: pcommon.AutomaticalMinWorkerCount(
			trServe.Channels()+s.cfg.Mapping.TotalCapacity(), 128),
		MaxWorkerIdle:     s.cfg.TransceiverIdleTimeout * 10,
		JobReceiveTimeout: s.cfg.TransceiverInitialTimeout,
	}).Serve()

	if runnerServeErr != nil {
		return runnerServeErr
	}

	s.runner = runner

	// Start all servers
	shb := &common.SharedBuffer{
		Buffer: make([]byte, 4096*trServe.Connections()),
		Size:   4096}

	s.servers = make([]network.Serving, len(s.cfg.Mapping))
	startedServer := 0

	for mIdx := range s.cfg.Mapping {
		var serving network.Serving
		var serveErr error

		switch s.cfg.Mapping[mIdx].Protocol {
		case network.TCP:
			serving, serveErr = server.New(tcplistener.New(
				s.cfg.Mapping[mIdx].Interface,
				s.cfg.Mapping[mIdx].Port,
				s.cfg.TransceiverInitialTimeout,
				tcpconn.Wrap,
			), tcpHandler{
				mapper:      s.cfg.Mapping[mIdx].ID,
				runner:      s.runner,
				shb:         shb,
				transceiver: s.transceiver,
				timeout:     s.cfg.TransceiverIdleTimeout,
			}, s.log.Context(strconv.FormatUint(
				uint64(s.cfg.Mapping[mIdx].ID), 10)+" ("+
				s.cfg.Mapping[mIdx].Protocol.String()+" "+
				net.JoinHostPort(s.cfg.Mapping[mIdx].Interface.String(),
					strconv.FormatUint(
						uint64(s.cfg.Mapping[mIdx].Port), 10))+")",
			), s.runner, server.Config{
				AcceptErrorWait: 300 * time.Millisecond,
				MaxConnections:  s.cfg.Mapping[mIdx].Capacity,
			}).Serve()

		case network.UDP:
			serving, serveErr = server.New(udplistener.New(
				s.cfg.Mapping[mIdx].Interface,
				s.cfg.Mapping[mIdx].Port,
				s.cfg.TransceiverIdleTimeout,
				s.cfg.Mapping[mIdx].Capacity,
				make([]byte, 4096),
				udpconn.Wrap,
			), udpHandler{
				mapper:      s.cfg.Mapping[mIdx].ID,
				runner:      s.runner,
				shb:         shb,
				transceiver: s.transceiver,
				timeout:     s.cfg.TransceiverIdleTimeout,
			}, s.log.Context(strconv.FormatUint(
				uint64(s.cfg.Mapping[mIdx].ID), 10)+" ("+
				s.cfg.Mapping[mIdx].Protocol.String()+" "+
				net.JoinHostPort(s.cfg.Mapping[mIdx].Interface.String(),
					strconv.FormatUint(
						uint64(s.cfg.Mapping[mIdx].Port), 10))+")",
			), s.runner, server.Config{
				AcceptErrorWait: 300 * time.Millisecond,
				MaxConnections:  s.cfg.Mapping[mIdx].Capacity,
			}).Serve()

		default:
			return ErrUnknownNetProtoType
		}

		if serveErr != nil {
			s.log.Errorf("Failed to boot up server for mapper %d on \"%s\"",
				s.cfg.Mapping[mIdx].ID, net.JoinHostPort(
					s.cfg.Mapping[mIdx].Interface.String(),
					strconv.FormatUint(uint64(
						s.cfg.Mapping[mIdx].Port), 10)))

			return serveErr
		}

		s.servers[startedServer] = serving

		s.log.Infof("Serving %s mapper %d on \"%s\"",
			s.cfg.Mapping[mIdx].Protocol.String(),
			s.cfg.Mapping[mIdx].ID,
			serving.Listening())

		startedServer++
	}

	s.log.Infof("Mapper is ready")

	return nil
}

func (s *mapper) Unspawn() error {
	s.log.Infof("Closing")

	// Close transceiver
	if s.transceiver != nil {
		transErr := s.transceiver.Close()

		if transErr != nil {
			return transErr
		}
	}

	if s.trReadTimeoutTicker == nil {
		s.trReadTimeoutTicker.Stop()
		s.trReadTimeoutTicker = nil
	}

	// Then, close all servers
	for sIdx := range s.servers {
		if s.servers[sIdx] == nil {
			continue
		}

		s.servers[sIdx].Close()
		s.servers[sIdx] = nil
	}

	// Close runner at last
	if s.runner != nil {
		runnerErr := s.runner.Close()

		if runnerErr != nil {
			return runnerErr
		}
	}

	s.log.Infof("Server is closed")

	s.unspawnNotifier <- struct{}{}

	return nil
}
