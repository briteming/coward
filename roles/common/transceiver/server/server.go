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

package server

import (
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/ticker"
	"github.com/reinit/coward/roles/common/channel"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/common/transceiver/connection"
)

// Errors
var (
	ErrServerSelectingDisabledChannel = errors.New(
		"Selecting a disabled Channel")
)

// server implements transceiver.Server
type server struct {
	cfg        Config
	codec      transceiver.CodecBuilder
	timeTicker ticker.Requester
}

// New creates a new transceiver server
func New(
	codec transceiver.CodecBuilder,
	timeTicker ticker.Requester,
	cfg Config,
) transceiver.Server {
	return &server{
		cfg:        cfg,
		codec:      codec,
		timeTicker: timeTicker,
	}
}

// Handle handles a new client connection
func (s *server) Handle(
	l logger.Logger,
	conn network.Connection,
	commands command.Commands,
) error {
	log := l.Context("Transceiver")

	cc, ccErr := connection.Codec(conn, s.codec)

	if ccErr != nil {
		return ccErr
	}

	channelized := connection.Channelize(cc, s.timeTicker)
	channelized.Timeout(s.cfg.InitialTimeout)

	defer channelized.Shutdown()

	channels := channel.New(func(id channel.ID) fsm.Machine {
		sLog := log.Context(
			"Channel (" + strconv.FormatUint(uint64(id), 10) + ")")

		channelConn := channelized.For(id)

		return &handler{
			logger:     sLog,
			conn:       channelConn,
			commands:   commands,
			runningCmd: nil,
		}
	}, s.cfg.ConnectionChannels)

	defer func() {
		chDownErr := channels.Shutdown()

		if chDownErr == nil {
			log.Debugf("Closed")

			return
		}

		log.Warningf("Failed to shutdown Channel due to error: %s", chDownErr)
	}()

	log.Debugf("Serving")

	connectionTimeoutUpdated := false

	var tickErr error

	for {
		if s.cfg.ChannelDispatchDelay > 0 {
			time.Sleep(s.cfg.ChannelDispatchDelay)
		}

		channelID, machine, chGetErr := channelized.Dispatch(channels)

		if chGetErr != nil {
			connErr, isConnErr := chGetErr.(connection.Error)

			if !isConnErr || connErr.Get() != io.EOF {
				log.Warningf("Request dispatch has failed: %s", chGetErr)
			}

			// If dispatch has failed, no reason to keep the client
			// connected.
			// In fact, we have to disconnect the client as when dispatch
			// has failed, the Client <--> Server status sync will be
			// broken.
			return chGetErr
		}

		if !connectionTimeoutUpdated {
			channelized.Timeout(s.cfg.IdleTimeout)

			connectionTimeoutUpdated = true
		}

		if !machine.Running() {
			tickErr = machine.Bootup()

			if tickErr == nil {
				continue
			}

			log.Warningf("An error occurred when initializing request for "+
				"Channel %d: %s", channelID, tickErr)
		} else {
			tickErr = machine.Tick()

			if tickErr == nil {
				continue
			}

			sdErr := machine.Shutdown()

			if sdErr != nil {
				log.Warningf("An error occurred when handling request for "+
					"Channel %d: %s. And another error also occurred "+
					"during request shutdown: %s", channelID, tickErr, sdErr)
			} else {
				log.Warningf("An error occurred when handling request for "+
					"Channel %d: %s", channelID, tickErr)
			}
		}

		switch tErr := tickErr.(type) {
		case transceiver.CodecError:
			return tErr

		case connection.Error:
			return tErr
		}
	}
}
