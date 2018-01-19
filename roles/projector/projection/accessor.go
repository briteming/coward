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

package projection

import (
	"io"

	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
)

// Accessor is the accessing requesting client
type Accessor interface {
	Access() network.Connection
	Result(e error, retriable bool, resetProccessor bool, wait chan struct{})
	Runner() worker.Runner
	Proccessor(p Proccessor)
}

// Proccessor represents a handler that will handle Accessor
type Proccessor struct {
	Proccessor io.Closer
	Waiter     chan struct{}
}

// accessorResult running result of the accesser
type accessorResult struct {
	err             error
	retriable       bool
	resetProccessor bool
	wait            chan struct{}
}

// accessor implements Accessor
type accessor struct {
	access     network.Connection
	proccessor chan Proccessor
	result     chan accessorResult
	runner     worker.Runner
}

// Access returns the Connection of the Accessor
func (a accessor) Access() network.Connection {
	return a.access
}

// Result returns the running result of the Accessor
func (a accessor) Result(
	e error, retriable bool, resetProccessor bool, wait chan struct{}) {
	a.result <- accessorResult{
		err:             e,
		retriable:       retriable,
		resetProccessor: resetProccessor,
		wait:            wait,
	}
}

// Runner returns the runner for current Accessor
func (a accessor) Runner() worker.Runner {
	return a.runner
}

// Proccessor send current Proccessor back to requesting Accessor
func (a accessor) Proccessor(p Proccessor) {
	a.proccessor <- p
}
