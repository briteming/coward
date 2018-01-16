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
	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/mapper/common"
	proxycommon "github.com/reinit/coward/roles/proxy/common"
)

type udp struct {
	mapper proxycommon.MapID
	log    logger.Logger
	relay  relay.Relay
	cancel <-chan struct{}
}

// UDP returns a new UDP request builder
func UDP(
	mapper proxycommon.MapID,
	client network.Connection,
	runner worker.Runner,
	shb *common.SharedBuffer,
) transceiver.RequestBuilder {
	return func(
		id transceiver.ConnectionID,
		conn rw.ReadWriteDepleteDoner,
		log logger.Logger,
	) fsm.Machine {
		return tcp{
			log: log,
			relay: relay.New(log, runner, conn, shb.Select(id), udpRelay{
				mapper: mapper,
				client: client,
			}, make([]byte, 4096)),
			cancel: client.Closed(),
		}
	}
}

func (c udp) Bootup() (fsm.State, error) {
	bootErr := c.relay.Bootup(c.cancel)

	if bootErr != nil {
		return nil, bootErr
	}

	return c.tick, nil
}

func (c udp) tick(f fsm.FSM) error {
	tErr := c.relay.Tick()

	if tErr != nil {
		return tErr
	}

	if !c.relay.Running() {
		return f.Shutdown()
	}

	return nil
}

func (c udp) Shutdown() error {
	c.relay.Close()

	return nil
}
