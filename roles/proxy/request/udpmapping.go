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
	"io"
	"net"
	"time"

	"github.com/reinit/coward/common/corunner"
	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/proxy/common"
)

// Errors
var (
	ErrUDPMappingNotFound = errors.New(
		"UDP Mapping not found")
)

// UDPMapping UDP Mapping request
type UDPMapping struct {
	Runner      corunner.Runner
	Buffer      []byte
	Cancel      <-chan struct{}
	LocalAddr   net.Addr
	DialTimeout time.Duration
	Mapping     common.Mapping
}

type udpMapping struct {
	mapping     common.Mapping
	buf         []byte
	localAddr   net.Addr
	dialTimeout time.Duration
	rw          rw.ReadWriteDepleteDoner
	runner      corunner.Runner
	relay       relay.Relay
	cancel      <-chan struct{}
}

// ID returns current Request ID
func (c UDPMapping) ID() command.ID {
	return UDPCommandTransport
}

// New creates a new request context
func (c UDPMapping) New(rw rw.ReadWriteDepleteDoner) fsm.Machine {
	return &udpMapping{
		mapping:     c.Mapping,
		buf:         c.Buffer,
		localAddr:   c.LocalAddr,
		dialTimeout: c.DialTimeout,
		rw:          rw,
		runner:      c.Runner,
		relay:       nil,
		cancel:      c.Cancel,
	}
}

func (u *udpMapping) Bootup() (fsm.State, error) {
	_, rErr := io.ReadFull(u.rw, u.buf[:1])

	if rErr != nil {
		u.rw.Done()

		return nil, rErr
	}

	u.rw.Done()

	mapped, mappedErr := u.mapping.Get(common.MapID(u.buf[0]))

	if mappedErr != nil {
		u.rw.Write([]byte{UDPRespondMappingNotFound})

		return nil, mappedErr
	}

	if mapped.Protocol != network.UDP {
		u.rw.Write([]byte{UDPRespondMappingNotFound})

		return nil, ErrUDPMappingNotFound
	}

	u.relay = relay.New(u.runner, u.rw, u.buf, &udpMappingRelay{
		localAddr:      u.localAddr,
		resolveTimeout: u.dialTimeout,
		mapped:         mapped,
		listenIP:       nil,
	}, make([]byte, 4096))

	bootupErr := u.relay.Bootup(u.cancel)

	if bootupErr != nil {
		return nil, bootupErr
	}

	return u.tick, nil
}

func (u *udpMapping) tick(f fsm.FSM) error {
	tErr := u.relay.Tick()

	if tErr != nil {
		return tErr
	}

	if !u.relay.Running() {
		return f.Shutdown()
	}

	return nil
}

func (u *udpMapping) Shutdown() error {
	u.relay.Close()

	return nil
}
