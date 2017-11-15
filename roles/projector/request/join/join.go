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

package join

import (
	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/command"
	"github.com/reinit/coward/roles/projector/projection"
	"github.com/reinit/coward/roles/projector/request"
)

// join Join request handler
type join struct {
	logger     logger.Logger
	cfg        Config
	runner     worker.Runner
	registered registerations
}

// New creates a new join request
func New(
	projections projection.Projections,
	runner worker.Runner,
	logger logger.Logger,
	cfg Config,
) command.Command {
	return join{
		logger: logger,
		cfg:    cfg,
		runner: runner,
		registered: registerations{
			projections: projections,
			receivers: make(
				map[projection.ID]registeration, projection.MaxID),
		},
	}
}

// ID returns the ID of current request
func (j join) ID() command.ID {
	return request.RequestCommandJoin
}

// New creates a new request processor
func (j join) New(rw rw.ReadWriteDepleteDoner) fsm.Machine {
	return &processor{
		logger:                      j.logger,
		cfg:                         j.cfg,
		runner:                      j.runner,
		registered:                  j.registered,
		rw:                          rw,
		currentProjectionID:         0,
		currentReceiveResult:        nil,
		currentReceiver:             nil,
		currentReceivedAccessorChan: make(chan projection.Accessor, 1),
		currentReceivedAccessor: projection.Accessor{
			Access: nil, Error: nil},
		currentRemotePingTimer: make(chan timer.Stopper, 1),
		currentRelay:           nil,
		receivingCloser:        make(chan struct{}),
	}
}
