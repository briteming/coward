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
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/relay"
)

// Errors
var (
	ErrTCPLocalAccessDeined = errors.New(
		"Local access deined")

	ErrTCPInvalidTimeout = errors.New(
		"Invalid TCP dial timeout")
)

// Respond ID
const (
	TCPRespondOK              = 0x00
	TCPRespondUnreachable     = 0x01
	TCPRespondGeneralError    = 0x03
	TCPRespondAccessDeined    = 0x04
	TCPRespondMappingNotFound = 0x05
	TCPRespondBadRequest      = 0x06
)

// TCP request
type TCP struct {
	Logger            logger.Logger
	Runner            worker.Runner
	Buffer            []byte
	DialTimeout       time.Duration
	ConnectionTimeout time.Duration
	Cancel            <-chan struct{}
	NoLocalAccess     bool
}

type tcp struct {
	logger            logger.Logger
	buf               []byte
	dialTimeout       time.Duration
	connectionTimeout time.Duration
	runner            worker.Runner
	cancel            <-chan struct{}
	noLocalAccess     bool
	rw                rw.ReadWriteDepleteDoner
	relay             relay.Relay
}

// ID returns the Request ID
func (c TCP) ID() command.ID {
	panic("`ID` must be overrided")
}

// New creates a new request context
func (c TCP) New(rw rw.ReadWriteDepleteDoner, log logger.Logger) fsm.Machine {
	panic("`New` must be overrided")
}

// Bootup starts request
func (c *tcp) Bootup() (fsm.State, error) {
	panic("`Bootup` must be overrided")
}

func (c *tcp) tick(f fsm.FSM) error {
	tErr := c.relay.Tick()

	if tErr != nil {
		return tErr
	}

	if !c.relay.Running() {
		return f.Shutdown()
	}

	return nil
}

func (c *tcp) Shutdown() error {
	c.relay.Close()

	return nil
}
