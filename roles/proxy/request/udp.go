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
	"errors"
	"net"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/relay"
)

// UDP Respond ID
const (
	UDPRespondOK                    = 0x00
	UDPRespondGeneralError          = 0x01
	UDPRespondInvalidRequest        = 0x02
	UDPRespondFailedToListen        = 0x03
	UDPRespondMappingHostUnresolved = 0x04
	UDPRespondMappingNotFound       = 0x05
)

// UDP Send type
const (
	UDPSendIPv4 = 0x00
	UDPSendIPv6 = 0x01
	UDPSendHost = 0x02
)

// Errors
var (
	ErrUDPTransportFailedToGetLocalIP = errors.New(
		"Failed to get local IP")

	ErrUDPTransportUnknownAddressType = errors.New(
		"Unknown UDP address type")

	ErrUDPTransportInvalidAddressData = errors.New(
		"Invalid UDP address data")
)

// UDP Request
type UDP struct {
	Logger    logger.Logger
	Runner    worker.Runner
	Buffer    []byte
	Cancel    <-chan struct{}
	LocalAddr net.Addr
}

type udp struct {
	logger logger.Logger
	runner worker.Runner
	cancel <-chan struct{}
	rw     rw.ReadWriteDepleteDoner
	relay  relay.Relay
}

// ID returns current Request ID
func (c UDP) ID() command.ID {
	return UDPCommandDelegate
}

// New creates a new request context
func (c UDP) New(rw rw.ReadWriteDepleteDoner) fsm.Machine {
	return udp{
		runner: c.Runner,
		cancel: c.Cancel,
		rw:     rw,
		relay: relay.New(c.Logger, c.Runner, rw, c.Buffer, &udpRelay{
			localAddr: c.LocalAddr,
			listenIP:  nil,
		}, make([]byte, 4096)),
	}
}

func (u udp) Bootup() (fsm.State, error) {
	defer u.rw.Done()

	bootupErr := u.relay.Bootup(u.cancel)

	if bootupErr != nil {
		return nil, bootupErr
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
