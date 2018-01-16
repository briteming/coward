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
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/worker"
)

// runerCall calling information of the runner call
type runerCall struct {
	Logger logger.Logger
	Job    worker.Job
	Result chan error
}

// runner implements worker.Run, but can only run one job
// It been built for run a job in current local routine, so we can save some
// memories
type runner struct {
	callReceive chan runerCall
}

// Run add and execute the job
func (r runner) Run(
	l logger.Logger, j worker.Job, cancel <-chan struct{}) (chan error, error) {
	callInfo := runerCall{
		Logger: l,
		Job:    j,
		Result: make(chan error, 1),
	}

	r.callReceive <- callInfo

	return callInfo.Result, nil
}

// RunWait add, execute the job and wait for the result
func (r runner) RunWait(
	l logger.Logger, j worker.Job, cancel <-chan struct{}) error {
	result, joinErr := r.Run(l, j, cancel)

	if joinErr != nil {
		return joinErr
	}

	return <-result
}

// Close shutdown current runner
func (r runner) Close() error {
	close(r.callReceive)

	return nil
}
