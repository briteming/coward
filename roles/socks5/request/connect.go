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

package request

import (
	"errors"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/socks5/common"
)

// Errors
var (
	ErrConnectInvalidAddressType = errors.New(
		"Invalid Socks5 address type")

	ErrConnectInitialRespondUnknownError = errors.New(
		"Unknown error for initial respond")

	ErrConnectInitialRelayFailed = errors.New(
		"Remote Relay has failed to initialize")

	ErrConnectInitialRespondGeneralError = errors.New(
		"Some error happened at the remote cause the request to fail")

	ErrConnectInitialRespondAccessDeined = errors.New(
		"Remote has deined the request")

	ErrConnectInitialRespondTargetUnreachable = errors.New(
		"Remote has failed to connect to the specified host")

	ErrConnectInitialFailedBadRequest = errors.New(
		"Remote has failed to initialize due to an invalid request")

	ErrConnectInvalidExchangeRespond = errors.New(
		"Invalid exchange respond")
)

type connect struct {
	log    logger.Logger
	relay  relay.Relay
	cancel <-chan struct{}
}

// Connect returns a Connect request builder
func Connect(
	client network.Connection,
	addr common.Address,
	runner worker.Runner,
	shb *common.SharedBuffers,
	requestTimeout time.Duration,
) transceiver.BalancedRequestBuilder {
	return func(
		cID transceiver.ClientID,
		id transceiver.ConnectionID,
		conn rw.ReadWriteDepleteDoner,
		log logger.Logger,
	) fsm.Machine {
		return connect{
			log: log,
			relay: relay.New(
				log, runner, conn, shb.For(cID).Select(id), connectRelay{
					client:         client,
					addr:           addr,
					requestTimeout: requestTimeout,
				}, make([]byte, 4096)),
			cancel: client.Closed(),
		}
	}
}

func (c connect) Bootup() (fsm.State, error) {
	bootErr := c.relay.Bootup(c.cancel)

	if bootErr != nil {
		return nil, bootErr
	}

	return c.tick, nil
}

func (c connect) tick(f fsm.FSM) error {
	tErr := c.relay.Tick()

	if tErr != nil {
		return tErr
	}

	if !c.relay.Running() {
		return f.Shutdown()
	}

	return nil
}

func (c connect) Shutdown() error {
	c.relay.Close()

	return nil
}
