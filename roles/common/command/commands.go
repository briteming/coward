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

package command

import (
	"errors"
	"fmt"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
)

// Errors
var (
	ErrRegisterAlreadyExisted = errors.New(
		"Command already existed")

	ErrSelectCommandUndefined = errors.New(
		"Selecting an undefined Command")
)

// ID represents a Command ID
type ID uint8

// Commands is the command proccesser
type Commands interface {
	Select(id ID) (Command, error)
}

// Command is a command
type Command interface {
	ID() ID
	New(rw.ReadWriteDepleteDoner) fsm.Machine
}

// Consts
const (
	MaxCommands = ID(64)
)

// commands implements Commands
type commands [MaxCommands]Command

// New creates a new Commands
func New(cmds ...Command) Commands {
	c := commands{}

	for cmdIdx := range cmds {
		if cmds[cmdIdx].ID() >= MaxCommands {
			panic(fmt.Sprintf("Command ID cannot be greater than %d",
				MaxCommands))
		}

		if c[cmds[cmdIdx].ID()] != nil {
			panic(fmt.Sprintf("Command %d already existed", cmds[cmdIdx].ID()))
		}

		c[cmds[cmdIdx].ID()] = cmds[cmdIdx]
	}

	return c
}

// Select selects a existing command
func (c commands) Select(id ID) (Command, error) {
	if id >= MaxCommands {
		return nil, ErrSelectCommandUndefined
	}

	if c[id] == nil {
		return nil, ErrSelectCommandUndefined
	}

	return c[id], nil
}
