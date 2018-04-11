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
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/proxy/common"
	"github.com/reinit/coward/roles/proxy/request"
)

type handler struct {
	transceiver transceiver.Server
	runner      worker.Runner
	mapping     common.Mapping
	cfg         Config
}

type client struct {
	conn        network.Connection
	logger      logger.Logger
	mapping     common.Mapping
	transceiver transceiver.Server
	runner      worker.Runner
	cfg         Config
}

func (d handler) New(
	c network.Connection,
	l logger.Logger,
) (network.Client, error) {
	return client{
		conn:        c,
		logger:      l,
		mapping:     d.mapping,
		transceiver: d.transceiver,
		runner:      d.runner,
		cfg:         d.cfg,
	}, nil
}

func (d client) Serve() error {
	buf := [4096]byte{}

	return d.transceiver.Handle(
		d.logger,
		d.conn,
		command.New(
			request.TCPIPv4{
				TCP: request.TCP{
					Runner:            d.runner,
					Buffer:            buf[:],
					DialTimeout:       d.cfg.InitialTimeout,
					ConnectionTimeout: d.cfg.IdleTimeout,
					Cancel:            d.conn.Closed(),
					NoLocalAccess:     true,
				},
			},
			request.TCPIPv6{
				TCP: request.TCP{
					Runner:            d.runner,
					Buffer:            buf[:],
					DialTimeout:       d.cfg.InitialTimeout,
					ConnectionTimeout: d.cfg.IdleTimeout,
					Cancel:            d.conn.Closed(),
					NoLocalAccess:     true,
				},
			},
			request.TCPHost{
				TCP: request.TCP{
					Runner:            d.runner,
					Buffer:            buf[:],
					DialTimeout:       d.cfg.InitialTimeout,
					ConnectionTimeout: d.cfg.IdleTimeout,
					Cancel:            d.conn.Closed(),
					NoLocalAccess:     true,
				},
			},
			request.TCPMapping{
				TCP: request.TCP{
					Runner:            d.runner,
					Buffer:            buf[:],
					DialTimeout:       d.cfg.InitialTimeout,
					ConnectionTimeout: d.cfg.IdleTimeout,
					Cancel:            d.conn.Closed(),
					NoLocalAccess:     false,
				},
				Mapping: d.mapping,
			},
			request.UDP{
				Runner:    d.runner,
				Buffer:    buf[:],
				Cancel:    d.conn.Closed(),
				LocalAddr: d.conn.LocalAddr(),
			},
			request.UDPMapping{
				Runner:      d.runner,
				Buffer:      buf[:],
				Cancel:      d.conn.Closed(),
				LocalAddr:   d.conn.LocalAddr(),
				DialTimeout: d.cfg.InitialTimeout,
				Mapping:     d.mapping,
			},
		),
	)
}
