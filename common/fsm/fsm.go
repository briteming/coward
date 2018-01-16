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

package fsm

import "errors"

// Errors
var (
	ErrBootupMustChangeState = errors.New(
		"Machine state must be changed after Bootup")

	ErrTickInvalidState = errors.New(
		"Invalid state for Ticking")

	ErrShutdownNotRunning = errors.New(
		"Can't Shutdown a machine that isn't running")

	ErrBootupAlreadyRunning = errors.New(
		"Can't Bootup a machine that already running")
)

// State is the Machine state
type State func(FSM) error

// Machine is a State machine
type Machine interface {
	Bootup() (State, error)
	Shutdown() error
}

// FSM is the Machine state manager
type FSM interface {
	Switch(State)
	SwitchTick(State) error
	Running() bool
	Tick() error
	Bootup() error
	Shutdown() error
}

// fsm implements FSM
type fsm struct {
	machine Machine
	current State
}

// New creates a new FSM
func New(m Machine) FSM {
	return &fsm{
		machine: m,
		current: nil,
	}
}

// Switch to a new state
func (f *fsm) Switch(s State) {
	f.current = s
}

// SwitchTick Switch to a new state and tick immediately
func (f *fsm) SwitchTick(s State) error {
	f.Switch(s)

	return f.Tick()
}

// Tick Ticks on current state
func (f *fsm) Tick() error {
	if f.current == nil {
		return ErrTickInvalidState
	}

	return f.current(f)
}

// Running returns the running status of current FSM
func (f *fsm) Running() bool {
	return f.current != nil
}

// Bootup starts the machine
func (f *fsm) Bootup() error {
	if f.Running() {
		return ErrBootupAlreadyRunning
	}

	switchTo, bootupErr := f.machine.Bootup()

	if bootupErr != nil {
		return bootupErr
	}

	if switchTo == nil {
		return ErrBootupMustChangeState
	}

	f.Switch(switchTo)

	return nil
}

// Shutdown closes the machine
func (f *fsm) Shutdown() error {
	if !f.Running() {
		return ErrShutdownNotRunning
	}

	shutdownErr := f.machine.Shutdown()

	if shutdownErr != nil {
		return shutdownErr
	}

	f.current = nil

	return nil
}
