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

package channel

import (
	"errors"
	"math"

	"github.com/reinit/coward/common/fsm"
)

// Errors
var (
	ErrGetUnregisteredChannels = errors.New(
		"Selecting an unregistered Channel")

	ErrIdleNoIdle = errors.New(
		"No Channels is idle")
)

// ID represent a Channel ID
type ID uint8

// Byte convert ID to a single byte
func (i ID) Byte() byte {
	return byte(i)
}

// HandlerBuilder builds a FSM Machine
type HandlerBuilder func(id ID) fsm.Machine

// Channels is the a set of FSMs that will be used to proccess
// different requests
type Channels interface {
	Get(id ID) (fsm.FSM, error)
	All(callback func(id ID, m fsm.FSM) (bool, error)) error
	Idle() (id ID, m fsm.FSM, err error)
	Size() uint8
	Shutdown() error
}

// Consts
const (
	MaxChannels = ID(math.MaxUint8)
)

// channels implements Channels
type channels struct {
	channels []fsm.FSM
	size     uint8
}

// New creates a new Channels
func New(p HandlerBuilder, size uint8) Channels {
	c := channels{
		channels: make([]fsm.FSM, size),
		size:     size,
	}

	for cIdx := range c.channels {
		c.channels[cIdx] = fsm.New(p(ID(cIdx)))
	}

	return c
}

func (c channels) Size() uint8 {
	return c.size
}

// Get retrieves a Channels FSM from channel register
func (c channels) Get(id ID) (fsm.FSM, error) {
	if id >= ID(c.size) {
		return nil, ErrGetUnregisteredChannels
	}

	if c.channels[id] == nil {
		return nil, ErrGetUnregisteredChannels
	}

	return c.channels[id], nil
}

// All fetchs all available FSM in channel
func (c channels) All(callback func(id ID, m fsm.FSM) (bool, error)) error {
	for v := range c.channels {
		goNext, callbackErr := callback(ID(v), c.channels[v])

		if callbackErr != nil {
			return callbackErr
		}

		if !goNext {
			break
		}
	}

	return nil
}

// Idle returns a idle channel
func (c channels) Idle() (id ID, m fsm.FSM, err error) {
	for v := range c.channels {
		if c.channels[v].Running() {
			continue
		}

		return ID(v), c.channels[v], nil
	}

	return 0, nil, ErrIdleNoIdle
}

// Shutdown turn off all Channels FSM
func (c channels) Shutdown() error {
	for v := range c.channels {
		if !c.channels[v].Running() {
			continue
		}

		sErr := c.channels[v].Shutdown()

		if sErr != nil {
			return sErr
		}
	}

	return nil
}
