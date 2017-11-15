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

package socks5

import (
	"time"

	"github.com/reinit/coward/common/corunner"
	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/network/server"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/socks5/common"
	"github.com/reinit/coward/roles/socks5/request"
)

type handler struct {
	cfg           Config
	runner        corunner.Runner
	shb           *common.SharedBuffers
	transceiver   transceiver.Balanced
	negoTimeout   time.Duration
	timeout       time.Duration
	authenticator Authenticator
}

type client struct {
	conn          network.Connection
	logger        logger.Logger
	cfg           Config
	transceiver   transceiver.Balanced
	negoTimeout   time.Duration
	timeout       time.Duration
	shb           *common.SharedBuffers
	authenticator Authenticator
	runner        corunner.Runner
}

func (d handler) New(
	c network.Connection,
	l logger.Logger,
) (server.Client, error) {
	return client{
		conn:          c,
		logger:        l,
		cfg:           d.cfg,
		transceiver:   d.transceiver,
		negoTimeout:   d.negoTimeout,
		timeout:       d.timeout,
		shb:           d.shb,
		authenticator: d.authenticator,
		runner:        d.runner,
	}, nil
}

func (d client) Serve() error {
	d.logger.Infof("Serving")
	defer d.logger.Infof("Closed")

	// Init negotiator
	nego := &negotiator{
		cfg:                    d.cfg,
		conn:                   d.conn,
		runner:                 d.runner,
		shb:                    d.shb,
		authenticator:          d.authenticator,
		selectedCMD:            0,
		selectedAddress:        common.Address{},
		selectedRequestBuilder: nil,
	}
	negoFSM := fsm.New(nego)

	// Give it a shorter timeout first
	d.conn.SetTimeout(d.negoTimeout)

	bootErr := negoFSM.Bootup()

	if bootErr != nil {
		d.logger.Warningf("Failed to start negotiation due to error: %s",
			bootErr)

		return bootErr
	}

	for {
		tErr := negoFSM.Tick()

		if tErr != nil {
			d.logger.Warningf("Negotiation has failed due to error: %s", tErr)

			return tErr
		}

		if negoFSM.Running() {
			continue
		}

		break
	}

	destName, req, reqBuildErr := nego.Build()

	if reqBuildErr != nil {
		d.logger.Warningf("Failed to build request due to error: %s",
			reqBuildErr)

		return reqBuildErr
	}

	// Change to a longer timeout
	d.conn.SetTimeout(d.timeout)

	reqErr := d.transceiver.Request(
		d.conn.RemoteAddr(), destName, req, d.conn.Closed())

	switch reqErr {
	case nil:
		d.logger.Debugf("Request completed")
		return nil

	case request.ErrConnectInvalidAddressType:
		rw.WriteFull(d.conn, []byte{
			0x05, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	case request.ErrConnectInitialRespondUnknownError:
		fallthrough
	case request.ErrConnectInitialRespondGeneralError:
		rw.WriteFull(d.conn, []byte{
			0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	case request.ErrConnectInitialRespondAccessDeined:
		rw.WriteFull(d.conn, []byte{
			0x05, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	case request.ErrConnectInitialRespondTargetUnreachable:
		rw.WriteFull(d.conn, []byte{
			0x05, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	case request.ErrUDPServerInvalidLocalAddr:
		rw.WriteFull(d.conn, []byte{
			0x05, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	case request.ErrUDPServerFailedToListen:
		fallthrough
	case request.ErrUDPInvalidRequest:
		fallthrough
	case request.ErrUDPServerRelayFailed:
		fallthrough
	case request.ErrUDPUnknownError:
		rw.WriteFull(d.conn, []byte{
			0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	}

	d.logger.Warningf("Request has failed: %s", reqErr)

	return reqErr
}
