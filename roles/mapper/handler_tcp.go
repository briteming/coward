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

package mapper

import (
	"time"

	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/mapper/common"
	"github.com/reinit/coward/roles/mapper/request"
	proxycommon "github.com/reinit/coward/roles/proxy/common"
)

type tcpHandler struct {
	mapper      proxycommon.MapID
	cfg         Config
	runner      worker.Runner
	shb         *common.SharedBuffer
	transceiver transceiver.Requester
	timeout     time.Duration
}

type tcpClient struct {
	mapper      proxycommon.MapID
	conn        network.Connection
	logger      logger.Logger
	cfg         Config
	transceiver transceiver.Requester
	timeout     time.Duration
	shb         *common.SharedBuffer
	runner      worker.Runner
}

func (d tcpHandler) New(
	c network.Connection,
	l logger.Logger,
) (network.Client, error) {
	return tcpClient{
		mapper:      d.mapper,
		conn:        c,
		logger:      l,
		cfg:         d.cfg,
		transceiver: d.transceiver,
		timeout:     d.timeout,
		shb:         d.shb,
		runner:      d.runner,
	}, nil
}

func (d tcpClient) Serve() error {
	d.logger.Infof("Serving")
	defer d.logger.Infof("Closed")

	metering := &meter{
		connection: timer.New(),
		request:    timer.New(),
	}

	d.conn.SetTimeout(d.timeout)

	_, reqErr := d.transceiver.Request(
		d.logger,
		request.TCP(d.mapper, d.conn, d.runner, d.shb),
		d.conn.Closed(), metering)

	if reqErr != nil {
		d.logger.Warningf("Request has failed: %s", reqErr)

		return reqErr
	}

	d.logger.Debugf("Request completed. Connection delay %s, request delay %s",
		metering.connection.Duration(), metering.request.Duration())

	return reqErr
}
