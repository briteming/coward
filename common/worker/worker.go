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

package worker

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reinit/coward/common/logger"
)

// Errors
var (
	ErrJobReceiveTimedout = errors.New(
		"Job Recieve timed out")

	ErrJobReceiveClosed = errors.New(
		"Job Recieve has been closed")

	ErrJobJoinCanceled = errors.New(
		"Job Join has been canceled")

	ErrAlreadyUp = errors.New(
		"Already serving")

	ErrAlreadyDown = errors.New(
		"Already closed")
)

// Consts
const (
	idleCheckTickDelay = 600 * time.Second
)

// Workers is a Go routine manager
type Workers interface {
	Serve() (Runner, error)
}

// Runner is a serving Jobs
type Runner interface {
	Run(log logger.Logger, j Job, cancel <-chan struct{}) (chan error, error)
	RunWait(log logger.Logger, j Job, cancel <-chan struct{}) error
	Close() error
}

// Job to run
type Job func(logger.Logger) error

// job dispatch data
type job struct {
	Logger logger.Logger
	Job    Job
	Result chan error
}

// workers implements Workers
type workers struct {
	log                   logger.Logger
	cfg                   Config
	booted                bool
	bootLock              sync.Mutex
	job                   chan job
	jobReceiveTicker      *time.Ticker
	workerID              chan uint32
	workerCount           uint32
	idle                  chan chan bool
	idleCheckTicker       *time.Ticker
	idleCheckTick         <-chan time.Time
	idleCheckChecking     chan struct{}
	idleNextCheck         time.Time
	idleMaxReleaseWorkers uint32
	shutdown              chan struct{}
	shutdownWait          sync.WaitGroup
}

// New creates a new Jobs
func New(log logger.Logger, cfg Config) Workers {
	return &workers{
		log:                   log.Context("Workers"),
		cfg:                   cfg,
		booted:                false,
		bootLock:              sync.Mutex{},
		job:                   make(chan job),
		jobReceiveTicker:      nil,
		workerID:              make(chan uint32, 1),
		workerCount:           0,
		idle:                  make(chan chan bool),
		idleCheckTicker:       nil,
		idleCheckTick:         nil,
		idleCheckChecking:     make(chan struct{}, 1),
		idleNextCheck:         time.Now().Add(cfg.MaxWorkerIdle),
		idleMaxReleaseWorkers: cfg.MinWorkers,
		shutdown:              make(chan struct{}, 1),
		shutdownWait:          sync.WaitGroup{},
	}
}

// worker is a Job worker
func (c *workers) worker(ready chan struct{}) {
	workerID := <-c.workerID
	workerID++
	c.workerID <- workerID

	workerName := "Worker (" + strconv.FormatUint(uint64(workerID), 10) + ")"
	log := c.log.Context(workerName)

	atomic.AddUint32(&c.workerCount, 1)

	defer func() {
		atomic.AddUint32(&c.workerCount, ^uint32(0))

		log.Debugf("Closed")

		c.shutdownWait.Done()
	}()

	idleExit := make(chan bool)

	log.Debugf("Serving")

	for {
		select {
		case ready <- struct{}{}:
			ready = nil

		case j := <-c.job:
			j.Result <- j.Job(j.Logger.Context(workerName))

		case d := <-c.shutdown:
			c.shutdown <- d

			return

		case c.idle <- idleExit:
			if !<-idleExit {
				continue
			}

			return

		case <-c.idleCheckTick:
			select {
			case c.idleCheckChecking <- struct{}{}:
				func() {
					defer func() {
						<-c.idleCheckChecking
					}()

					if time.Now().Before(c.idleNextCheck) {
						return
					}

					c.idleNextCheck = time.Now().Add(c.cfg.MaxWorkerIdle)

					idles := make([]chan bool, c.cfg.MaxWorkers)
					idleWorkers := uint32(0)

					for idleIdx := range idles {
						select {
						case idleInfo := <-c.idle:
							idles[idleIdx] = idleInfo
							idleWorkers++

						default:
						}

						break
					}

					if idleWorkers <= 0 || idleWorkers <= c.cfg.MinWorkers {
						for idleIdx := range idles[:idleWorkers] {
							idles[idleIdx] <- false
						}

						c.idleMaxReleaseWorkers = c.cfg.MinWorkers

						return
					}

					pendingWorkers := c.cfg.MinWorkers - idleWorkers

					if pendingWorkers > c.idleMaxReleaseWorkers {
						pendingWorkers = c.idleMaxReleaseWorkers

						c.idleMaxReleaseWorkers *= 2
					}

					for idleIdx := range idles[pendingWorkers:idleWorkers] {
						idles[idleIdx] <- false
					}

					for idleIdx := range idles[0:pendingWorkers] {
						idles[idleIdx] <- true
					}

					log.Debugf("Releasing %d Workers", pendingWorkers)
				}()

			default:
			}
		}
	}
}

// createWorkers creates worker
func (c *workers) createWorkers(num uint32) {
	remainFreeWorkers := c.cfg.MaxWorkers - atomic.LoadUint32(&c.workerCount)

	if remainFreeWorkers <= 0 {
		return
	}

	if num > remainFreeWorkers {
		num = remainFreeWorkers
	}

	c.idleCheckChecking <- struct{}{}

	defer func() {
		<-c.idleCheckChecking
	}()

	ready := make(chan struct{}, num)

	for i := uint32(0); i < num; i++ {
		c.shutdownWait.Add(1)

		go c.worker(ready)
	}

	for range ready {
		num--

		if num <= 0 {
			break
		}
	}

	close(ready)
}

// Serve start serve
func (c *workers) Serve() (Runner, error) {
	c.bootLock.Lock()
	defer c.bootLock.Unlock()

	if c.booted {
		return nil, ErrAlreadyUp
	}

	c.workerID <- 0

	c.idleCheckTicker = time.NewTicker(idleCheckTickDelay)
	c.jobReceiveTicker = time.NewTicker(c.cfg.JobReceiveTimeout)

	c.createWorkers(c.cfg.MinWorkers)

	c.booted = true

	return c, nil
}

// Close stop serve
func (c *workers) Close() error {
	c.bootLock.Lock()
	defer c.bootLock.Unlock()

	if !c.booted {
		return ErrAlreadyDown
	}

	c.idleCheckTicker.Stop()
	c.jobReceiveTicker.Stop()

	c.shutdown <- struct{}{}
	c.shutdownWait.Wait()
	<-c.shutdown

	<-c.workerID

	c.booted = false

	return nil
}

// Run run a Job in a routine, and return a error channel to receive it's
// running result
func (c *workers) Run(
	l logger.Logger,
	j Job,
	cancel <-chan struct{},
) (chan error, error) {
	newJob := job{
		Logger: l,
		Job:    j,
		Result: make(chan error, 1),
	}

	select {
	case c.job <- newJob:
		return newJob.Result, nil

	default:
		c.bootLock.Lock()
		defer c.bootLock.Unlock()

		c.createWorkers(c.cfg.MinWorkers)
	}

	timeout := time.Now().Add(c.cfg.JobReceiveTimeout)

	for {
		select {
		case c.job <- newJob:
			return newJob.Result, nil

		case <-cancel:
			return nil, ErrJobJoinCanceled

		case d := <-c.shutdown:
			c.shutdown <- d

			return nil, ErrJobReceiveClosed

		case <-c.jobReceiveTicker.C:
			if time.Now().Before(timeout) {
				continue
			}

			return nil, ErrJobReceiveTimedout
		}
	}
}

// RunWait run a Job in a routine, and wait for it to complete
func (c *workers) RunWait(
	l logger.Logger,
	j Job,
	cancel <-chan struct{},
) error {
	runResult, runErr := c.Run(l, j, cancel)

	if runErr != nil {
		return runErr
	}

	return <-runResult
}
