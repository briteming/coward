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

package request

import (
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

type udp struct {
	client network.Connection
	log    logger.Logger
	relay  relay.Relay
	cancel <-chan struct{}
}

// UDP returns a new UDP request builder
func UDP(
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
		return &udp{
			log: log,
			relay: relay.New(
				log, runner, conn, shb.For(cID).Select(id), &udpRelay{
					client:         client,
					addr:           addr,
					requestTimeout: requestTimeout,
					runner:         runner,
					cancel:         client.Closed(),
					udpConn:        nil,
					comfirmData:    nil,
				}, make([]byte, 4096)),
			cancel: client.Closed(),
		}
	}
}

func (u udp) Bootup() (fsm.State, error) {
	bootErr := u.relay.Bootup(u.cancel)

	if bootErr != nil {
		return nil, bootErr
	}

	return u.tick, nil
}

func (u udp) tick(f fsm.FSM) error {
	tErr := u.relay.Tick()

	if tErr != nil {
		return tErr
	}

	if !u.relay.Running() {
		return f.Shutdown()
	}

	return nil
}

func (u udp) Shutdown() error {
	u.relay.Close()

	return nil
}
