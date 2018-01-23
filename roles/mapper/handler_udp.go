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

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/mapper/common"
	"github.com/reinit/coward/roles/mapper/request"
	proxycommon "github.com/reinit/coward/roles/proxy/common"
)

type udpHandler struct {
	mapper      proxycommon.MapID
	cfg         Config
	runner      worker.Runner
	shb         *common.SharedBuffer
	transceiver transceiver.Requester
	timeout     time.Duration
	reqTimeout  time.Duration
}

type udpClient struct {
	mapper      proxycommon.MapID
	conn        network.Connection
	logger      logger.Logger
	cfg         Config
	transceiver transceiver.Requester
	timeout     time.Duration
	reqTimeout  time.Duration
	shb         *common.SharedBuffer
	runner      worker.Runner
}

func (d udpHandler) New(
	c network.Connection,
	l logger.Logger,
) (network.Client, error) {
	return udpClient{
		mapper:      d.mapper,
		conn:        c,
		logger:      l,
		cfg:         d.cfg,
		transceiver: d.transceiver,
		timeout:     d.timeout,
		reqTimeout:  d.reqTimeout,
		shb:         d.shb,
		runner:      d.runner,
	}, nil
}

func (d udpClient) Serve() error {
	d.logger.Infof("Serving")
	defer d.logger.Infof("Cleaned up")

	d.conn.SetTimeout(d.reqTimeout)

	metering := &meter{
		connection: timer.New(),
		request:    timer.New(),
	}

	_, reqErr := d.transceiver.Request(
		d.logger,
		request.UDP(d.mapper, d.conn, d.runner, d.timeout, d.shb),
		d.conn.Closed(), metering)

	if reqErr != nil {
		d.logger.Warningf("Request has failed: %s", reqErr)

		return reqErr
	}

	d.logger.Debugf("Request closed. Connection delay %s, request delay %s",
		metering.connection.Duration(), metering.request.Duration())

	return reqErr
}
