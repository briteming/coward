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

package server

import (
	"errors"
	"io"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/common/transceiver/connection"
)

// Errors
var (
	ErrHandlerFailedToReadCommandID = errors.New(
		"Failed to read Command ID")
)

type handler struct {
	logger     logger.Logger
	conn       connection.Virtual
	commands   command.Commands
	runningCmd fsm.FSM
}

func (h *handler) Bootup() (fsm.State, error) {
	cmdIDBuf := [1]byte{}

	rLen, rErr := io.ReadFull(h.conn, cmdIDBuf[:])

	if rErr != nil {
		h.conn.Done()

		return nil, ErrHandlerFailedToReadCommandID
	}

	if rLen != 1 {
		h.conn.Done()

		return nil, ErrHandlerFailedToReadCommandID
	}

	cmd, cmdSelectErr := h.commands.Select(command.ID(cmdIDBuf[0]))

	if cmdSelectErr != nil {
		h.conn.Done()

		h.logger.Debugf("Failed to select Command \"%d\" due to error: %s",
			cmdIDBuf[0], cmdSelectErr)

		return nil, cmdSelectErr
	}

	cmdRunner := fsm.New(cmd.New(h.conn, h.logger))

	// Notice the Bootup will continue reading the segment rather than
	// request other one. So don't clear the h.conn before Bootup is called.
	// And actually, don't clear it after as well. Because the Bootup must
	// clear the h.conn by itself
	bootupErr := cmdRunner.Bootup()

	if bootupErr != nil {
		return nil, bootupErr
	}

	h.runningCmd = cmdRunner

	h.logger.Debugf("Handling Command %d", cmd.ID())

	return h.run, nil
}

func (h *handler) run(f fsm.FSM) error {
	tErr := h.runningCmd.Tick()

	if tErr != nil {
		return tErr
	}

	if h.runningCmd.Running() {
		return nil
	}

	return f.Shutdown()
}

func (h *handler) Shutdown() error {
	if h.runningCmd == nil {
		return nil
	}

	runningCmd := h.runningCmd
	h.runningCmd = nil

	if !runningCmd.Running() {
		h.logger.Debugf("Request completed")
	} else {
		shutdownErr := runningCmd.Shutdown()

		if shutdownErr != nil {
			h.logger.Debugf("Request exited with error: %s", shutdownErr)

			return shutdownErr
		}

		h.logger.Debugf("Request exited")
	}

	return nil
}
