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
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/ticker"
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

// workerJob dispatch data
type workerJob struct {
	Logger logger.Logger
	Job    Job
	Result chan error
}

// workerClose signal
type workerClose struct {
	Closed chan struct{}
}

// workerCreate request
type workerCreate struct {
	Created chan struct{}
}

// workers implements Workers
type workers struct {
	log                   logger.Logger
	ticker                ticker.Requester
	cfg                   Config
	booted                bool
	bootLock              sync.Mutex
	job                   chan workerJob
	create                chan workerCreate
	idleMaxReleaseWorkers uint32
	shutdown              chan struct{}
	shutdownWait          sync.WaitGroup
}

// New creates a new Jobs
func New(log logger.Logger, tick ticker.Requester, cfg Config) Workers {
	return &workers{
		log:                   log.Context("Workers"),
		ticker:                tick,
		cfg:                   cfg,
		booted:                false,
		bootLock:              sync.Mutex{},
		job:                   make(chan workerJob),
		create:                make(chan workerCreate),
		idleMaxReleaseWorkers: cfg.MinWorkers,
		shutdown:              make(chan struct{}, 1),
		shutdownWait:          sync.WaitGroup{},
	}
}

// maintainer create and destroy workers
func (c *workers) maintainer(ready chan struct{}) {
	var shutdownReceive chan struct{}
	var createReceive chan workerCreate
	var releaseWait ticker.Wait

	log := c.log.Context("Maintainer")
	workingWorkers := uint32(0)
	lastWorkerID := uint32(0)
	maxWorkersRelease := c.cfg.MinWorkers
	nextRelease := time.Now().Add(c.cfg.MaxWorkerIdle)
	closeSignal := make(chan workerClose)
	quitNotify := make(chan struct{}, c.cfg.MaxWorkers)

	defer func() {
		if closeSignal != nil {
			close(closeSignal)
		}

		close(quitNotify)

		log.Debugf("Closed")

		c.shutdownWait.Done()
	}()

	releaseWaiter, releaseWaiterErr := c.ticker.Request(nextRelease)

	if releaseWaiterErr != nil {
		panic(fmt.Sprintf("Failed to initialize maintainer ticker"+
			" due to error: %s", releaseWaiterErr))
	}

	for {
		select {
		case ready <- struct{}{}:
			ready = nil

			shutdownReceive = c.shutdown
			createReceive = c.create
			releaseWait = releaseWaiter.Wait()

		case d := <-shutdownReceive:
			shutdownReceive <- d

			shutdownReceive = nil
			createReceive = nil

			releaseWait = nil
			releaseWaiter.Close()

			if workingWorkers > 0 {
				close(closeSignal)
				closeSignal = nil

				continue
			}

			return

		case cd := <-createReceive:
			maxWorkersRelease = c.cfg.MinWorkers
			nextRelease = time.Now().Add(c.cfg.MaxWorkerIdle)

			func(cd workerCreate) {
				defer func() {
					cd.Created <- struct{}{}
				}()

				wCanCreate := c.cfg.MaxWorkers - workingWorkers

				if wCanCreate <= 0 {
					return
				}

				wToCreate := c.cfg.MinWorkers

				if wToCreate > wCanCreate {
					wToCreate = wCanCreate
				}

				workerReady := make(chan struct{}, 1)
				wCreated := uint32(0)

				for wCreated < wToCreate {
					go c.worker(
						lastWorkerID,
						workerReady,
						quitNotify,
						closeSignal,
					)

					<-workerReady

					lastWorkerID++
					workingWorkers++
					wCreated++
				}

				log.Debugf("%d Workers has been created", wCreated)
			}(cd)

		case <-quitNotify:
			workingWorkers--

			if shutdownReceive != nil {
				continue
			}

			if workingWorkers > 0 {
				continue
			}

			return

		case <-releaseWait:
			releaseWaiter.Close()

			nextWaitTime := time.Now().Add(c.cfg.MaxWorkerIdle)
			releaseWaiter, releaseWaiterErr = c.ticker.Request(nextWaitTime)

			if releaseWaiterErr != nil {
				panic(fmt.Sprintf("Failed to initialize maintainer ticker "+
					"due to error: %s", releaseWaiterErr))
			}

			releaseWait = releaseWaiter.Wait()

			if time.Now().Before(nextRelease) {
				continue
			}

			nextRelease = nextWaitTime

			releasedWorkers := uint32(0)
			workerCloser := workerClose{
				Closed: make(chan struct{}),
			}
			stopReleasing := false

			for releasedWorkers < maxWorkersRelease && !stopReleasing {
				select {
				case closeSignal <- workerCloser:
					<-workerCloser.Closed

					releasedWorkers++

				default:
					stopReleasing = true
				}
			}

			if releasedWorkers <= 0 {
				maxWorkersRelease = c.cfg.MinWorkers

				continue
			}

			maxWorkersRelease += c.cfg.MinWorkers

			log.Debugf("%d Workers has been released", releasedWorkers)
		}
	}
}

// worker is a Job worker
func (c *workers) worker(
	workerID uint32,
	ready chan struct{},
	quitNotify chan struct{},
	closeSignal chan workerClose,
) {
	var jobReceive chan workerJob
	var closeSignalReceive chan workerClose

	workerName := "Worker (" + strconv.FormatUint(uint64(workerID), 10) + ")"
	log := c.log.Context(workerName)

	defer func() {
		quitNotify <- struct{}{}

		log.Debugf("Closed")
	}()

	log.Debugf("Serving")

	for {
		select {
		case ready <- struct{}{}:
			ready = nil

			jobReceive = c.job
			closeSignalReceive = closeSignal

		case j := <-jobReceive:
			j.Result <- j.Job(j.Logger.Context(workerName))

		case close, closeOK := <-closeSignalReceive:
			if !closeOK {
				return
			}

			close.Closed <- struct{}{}

			return
		}
	}
}

// Serve start serve
func (c *workers) Serve() (Runner, error) {
	c.bootLock.Lock()
	defer c.bootLock.Unlock()

	if c.booted {
		return nil, ErrAlreadyUp
	}

	c.shutdownWait.Add(1)

	ready := make(chan struct{}, 1)
	go c.maintainer(ready)
	<-ready

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

	c.shutdown <- struct{}{}
	c.shutdownWait.Wait()
	<-c.shutdown

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
	newJob := workerJob{
		Logger: l,
		Job:    j,
		Result: make(chan error, 1),
	}

	timeout := time.Now().Add(c.cfg.JobReceiveTimeout)
	joinWait, joinWaitReqErr := c.ticker.Request(timeout)

	if joinWaitReqErr != nil {
		return nil, joinWaitReqErr
	}

	defer joinWait.Close()

	select {
	case c.job <- newJob:
		return newJob.Result, nil

	default:
		jobCreate := workerCreate{
			Created: make(chan struct{}),
		}

		select {
		case c.create <- jobCreate:
			<-jobCreate.Created

		case <-joinWait.Wait():
			return nil, ErrJobReceiveTimedout
		}
	}

	select {
	case c.job <- newJob:
		return newJob.Result, nil

	case <-cancel:
		return nil, ErrJobJoinCanceled

	case d := <-c.shutdown:
		c.shutdown <- d

		return nil, ErrJobReceiveClosed

	case <-joinWait.Wait():
		return nil, ErrJobReceiveTimedout
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
