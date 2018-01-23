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
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/projector/projection"
	"github.com/reinit/coward/roles/projector/request/join"
)

type handler struct {
	transceiver transceiver.Server
	runner      worker.Runner
	projections projection.Projections
	minTimeout  uint16
	cfg         Config
}

type client struct {
	conn        network.Connection
	logger      logger.Logger
	transceiver transceiver.Server
	runner      worker.Runner
	projections projection.Projections
	minTimeout  uint16
	cfg         Config
}

func (d handler) New(
	c network.Connection,
	l logger.Logger,
) (network.Client, error) {
	return client{
		conn:        c,
		logger:      l,
		transceiver: d.transceiver,
		runner:      d.runner,
		projections: d.projections,
		minTimeout:  d.minTimeout,
		cfg:         d.cfg,
	}, nil
}

func (d client) Serve() error {
	closeNotify := make(chan struct{})
	defer close(closeNotify)

	buf := [4096]byte{}

	return d.transceiver.Handle(
		d.logger,
		d.conn,
		command.New(join.New(
			d.projections,
			d.conn,
			closeNotify,
			d.runner,
			d.logger,
			join.Config{
				ConnectionID:     d.conn.ID(),
				ConnectionDelay:  timer.Average(),
				Buffer:           buf[:],
				Timeout:          d.minTimeout,
				ClientTimeout:    d.cfg.IdleTimeout,
				ClientReqTimeout: d.cfg.InitialTimeout,
			},
		)),
	)
}
