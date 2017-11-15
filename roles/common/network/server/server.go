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

package server

import (
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/roles/common/network"
)

// Server represents a Server which will accept and handle incomming
// network.Connection
type Server interface {
	Serve() (Serving, error)
}

// Serving represents a running Server
type Serving interface {
	Listening() net.Addr
	Close() error
}

// accept data
type accept struct {
	Conn   network.Connection
	Result chan error
}

// kill data
type kill struct {
	Done chan struct{}
}

// serving contains information that describe a serving Server
type serving struct {
	Server   *server
	Accepter network.Acceptor
	Kills    []chan kill
}

// client information
type client struct {
	Conn   network.Connection
	Closed bool
}

// pick contains client information for worker to pick up
type pick struct {
	Conn  network.Connection
	Leave chan network.Connection
}

// minIdleCheckDelay delay of Idle check
const minIdleCheckDelay = 300 * time.Second

// occupation status
type occupation uint8

// defined occupation status
const (
	oDestroyed  occupation = 0
	oOccupied   occupation = 1
	oUnoccupied occupation = 2
)

// server implements Server
type server struct {
	cfg                       Config
	listener                  network.Listener
	Handler                   Handler
	logger                    logger.Logger
	pick                      chan pick
	accept                    chan accept
	serverCloser              chan struct{}
	serverCloseWaiter         sync.WaitGroup
	lastWorkerID              uint64
	workerStarted             uint32
	workerOccupied            uint32
	workerOccupy              chan occupation
	workerCreation            chan chan struct{}
	workerCloser              chan struct{}
	workerProducerCloser      chan struct{}
	workerIdle                chan struct{}
	workerCloseWaiter         sync.WaitGroup
	workerProducerCloseWaiter sync.WaitGroup
	acceptorCloser            chan struct{}
	acceptorCloseWaiter       sync.WaitGroup
	serving                   bool
	servingLock               sync.Mutex
}

// New creates a new Server
func New(
	listener network.Listener,
	Handler Handler,
	logger logger.Logger,
	cfg Config,
) Server {
	s := &server{
		cfg:                       cfg,
		listener:                  listener,
		Handler:                   Handler,
		logger:                    logger.Context("Server"),
		pick:                      make(chan pick),
		accept:                    make(chan accept),
		serverCloser:              make(chan struct{}, 1),
		serverCloseWaiter:         sync.WaitGroup{},
		lastWorkerID:              0,
		workerStarted:             0,
		workerOccupied:            0,
		workerOccupy:              make(chan occupation),
		workerCreation:            make(chan chan struct{}, 1),
		workerCloser:              make(chan struct{}, 1),
		workerProducerCloser:      make(chan struct{}),
		workerIdle:                make(chan struct{}),
		workerCloseWaiter:         sync.WaitGroup{},
		workerProducerCloseWaiter: sync.WaitGroup{},
		acceptorCloser:            make(chan struct{}, 1),
		acceptorCloseWaiter:       sync.WaitGroup{},
		serving:                   false,
		servingLock:               sync.Mutex{},
	}

	s.acceptorCloser <- struct{}{}

	return s
}

// producer produce and destroy idle Workers
func (a *server) producer(minWorkers uint32) {
	log := a.logger.Context("Producer")

	idleCheckTickDelay := minIdleCheckDelay

	if a.cfg.MaxWorkerIdle > idleCheckTickDelay {
		idleCheckTickDelay = a.cfg.MaxWorkerIdle
	}

	idleCheckTicker := time.NewTicker(idleCheckTickDelay)
	idleCheckTickChan := idleCheckTicker.C
	nextIdleCheck := time.Now().Add(idleCheckTickDelay)
	maxIdleRelease := minWorkers
	shutdown := false

	defer func() {
		idleCheckTicker.Stop()

		log.Debugf("Closed")

		a.workerProducerCloseWaiter.Done()
	}()

	for {
		select {
		case <-a.workerProducerCloser:
			if shutdown {
				idleCheckTickChan = nil

				return
			}

			shutdown = true

		case occupy := <-a.workerOccupy:
			switch occupy {
			case oDestroyed:
				a.workerStarted--

			case oOccupied:
				a.workerOccupied++

				if shutdown {
					idleCheckTickChan = nil

					continue
				}

				created := a.replenishWorkers(minWorkers)

				if created > 0 {
					a.workerStarted += created

					nextIdleCheck = time.Now().Add(idleCheckTickDelay)
					maxIdleRelease = minWorkers

					log.Debugf("%d new Workers has been created", created)
				}

			case oUnoccupied:
				a.workerOccupied--

				idleCheckTickChan = idleCheckTicker.C
			}

		case workerCreationDone := <-a.workerCreation:
			// Shouldn't happen here, but I did the check anyway
			if shutdown {
				idleCheckTickChan = nil
				workerCreationDone <- struct{}{}

				continue
			}

			created := a.replenishWorkers(minWorkers)

			if created > 0 {
				a.workerStarted += created

				nextIdleCheck = time.Now().Add(idleCheckTickDelay)
				maxIdleRelease = minWorkers

				log.Debugf("%d new Workers has been created", created)
			}

			workerCreationDone <- struct{}{}

		case <-idleCheckTickChan:
			if shutdown {
				idleCheckTickChan = nil

				continue
			}

			now := time.Now()

			if now.Before(nextIdleCheck) {
				continue
			}

			nextIdleCheck = now.Add(idleCheckTickDelay)

			pendingWorkers := a.workerStarted - a.workerOccupied

			if pendingWorkers <= minWorkers {
				idleCheckTickChan = nil

				continue
			}

			pendingWorkers -= minWorkers

			if pendingWorkers > maxIdleRelease {
				pendingWorkers = maxIdleRelease
			}

			idleWorkers := make([]struct{}, pendingWorkers)
			idleWorkerReleaseCount := 0

			for workerIdx := range idleWorkers {
				select {
				case i := <-a.workerIdle:
					idleWorkers[workerIdx] = i
					idleWorkerReleaseCount++

					continue

				default:
				}

				break
			}

			maxIdleRelease *= 2

			log.Debugf("Releasing %d idle Workers", idleWorkerReleaseCount)
		}
	}
}

// worker is the worker 'thread's
func (a *server) worker(id uint64, ready chan<- struct{}) error {
	log := a.logger.Context(
		"Worker (" + strconv.FormatUint(id, 10) + ")")

	defer func() {
		if ready != nil {
			close(ready)
		}

		log.Debugf("Closed")

		a.workerOccupy <- oDestroyed

		a.workerCloseWaiter.Done()
	}()

	for {
		select {
		case ready <- struct{}{}:
			// Tell the creater we're ready
			close(ready)

			ready = nil

		case close := <-a.workerCloser:
			a.workerCloser <- close

			return nil

		case a.workerIdle <- struct{}{}:
			log.Debugf("Idle, releasing")

			return nil

		case picked := <-a.pick:
			func(p pick) {
				a.workerOccupy <- oOccupied

				defer func() {
					a.workerOccupy <- oUnoccupied
					p.Leave <- p.Conn
				}()

				clientLogger := log.Context(p.Conn.RemoteAddr().String())

				client, clientErr := a.Handler.New(p.Conn, clientLogger)

				if clientErr != nil {
					clientLogger.Debugf(
						"Initialization has failed due to error: %s", clientErr)

					return
				}

				clientErr = client.Serve()

				if clientErr != nil {
					clientLogger.Debugf(
						"Stopped serving the client under error: %s", clientErr)

					return
				}
			}(picked)
		}
	}
}

// acceptor is a single thread that will dispatch incomming network.Connection
// to a random free worker
func (a *server) acceptor(id uint64, killChan chan kill) error {
	log := a.logger.Context("Acceptor (" + strconv.FormatUint(id, 10) + ")")
	leave := make(chan network.Connection)
	clients := make(map[network.ConnectionID]client, a.cfg.MaxWorkers)
	workerCreationWaiter := make(chan struct{})
	killed := false

	defer func() {
		log.Debugf("Closed")

		close(leave)
		close(workerCreationWaiter)
		close(killChan)

		a.acceptorCloseWaiter.Done()
	}()

	for {
		select {
		case close := <-a.acceptorCloser:
			a.acceptorCloser <- close

			return nil

		case acc := <-a.accept:
			if killed {
				acc.Result <- ErrAlreadyClosed

				continue
			}

			giveup := false
			workerCreationRequested := false

			for !giveup {
				select {
				case a.pick <- pick{Conn: acc.Conn, Leave: leave}:
					clients[acc.Conn.ID()] = client{
						Conn:   acc.Conn,
						Closed: false,
					}

					giveup = true

					acc.Result <- nil

					log.Debugf("New connection from \"%s\"",
						acc.Conn.RemoteAddr().String())

				default:
					if !workerCreationRequested {
						a.workerCreation <- workerCreationWaiter
						<-workerCreationWaiter

						workerCreationRequested = true
					} else {
						giveup = true

						acc.Result <- ErrAcceptorTooBusy
					}
				}
			}

		case leave := <-leave:
			remoteConnID := leave.ID()
			requireClose := !clients[remoteConnID].Closed

			delete(clients, remoteConnID)

			if requireClose {
				leave.Close()
			}

			log.Debugf("Connection to \"%s\" is closed",
				leave.RemoteAddr().String())

		case kil := <-killChan:
			killed = true

			for n, v := range clients {
				v.Conn.Close()
				v.Closed = true

				clients[n] = v

				log.Debugf("Connection to \"%s\" has been closed",
					v.Conn.RemoteAddr())
			}

			kil.Done <- struct{}{}
		}
	}
}

// serve accepts new connections
func (a *server) serve(listen network.Acceptor) {
	log := a.logger.Context("Server")

	defer func() {
		log.Debugf("Closed")

		a.serverCloseWaiter.Done()
	}()

	for {
		accepted, acceptErr := listen.Accept()

		if acceptErr != nil {
			select {
			case closer := <-a.serverCloser:
				a.serverCloser <- closer

				return

			default:
				log.Warningf("Failed to accept new connection "+
					"due to error: %s. Waiting for %s to continue",
					acceptErr, a.cfg.AcceptErrorWait)

				time.Sleep(a.cfg.AcceptErrorWait)
			}

			continue
		}

		joinErr := a.join(accepted)

		if joinErr == nil {
			continue
		}

		log.Warningf(
			"Failed to accept new connection from \"%s\" due to error: %s",
			accepted.RemoteAddr(), joinErr)

		accepted.Close()
	}
}

// accept picks up incomming network.Connection, and dispatch it to a
// free worker
func (a *server) join(conn network.Connection) error {
	acc := accept{
		Conn:   conn,
		Result: make(chan error),
	}

	defer close(acc.Result)

	select {
	case a.accept <- acc:
		return <-acc.Result

	case closer := <-a.acceptorCloser:
		a.acceptorCloser <- closer

		return ErrAlreadyClosed
	}
}

// startWorkers spawns workers
func (a *server) startWorkers(num uint32) {
	for wIndex := uint32(0); wIndex < num; wIndex++ {
		a.workerCloseWaiter.Add(1)

		ready := make(chan struct{})

		go a.worker(a.lastWorkerID, ready)

		<-ready

		a.lastWorkerID++

		if a.lastWorkerID >= math.MaxUint64 {
			a.lastWorkerID = 0
		}
	}
}

// replenishWorkers tries to create Workers until limitation
// has reached
func (a *server) replenishWorkers(minWorkers uint32) uint32 {
	remainerWorkers := a.cfg.MaxWorkers - a.workerStarted

	if remainerWorkers <= 0 {
		return 0
	}

	idleWorkers := a.workerStarted - a.workerOccupied

	if idleWorkers >= minWorkers {
		return 0
	}

	neededIdleWorkers := minWorkers

	if neededIdleWorkers > remainerWorkers {
		neededIdleWorkers = remainerWorkers
	}

	a.startWorkers(neededIdleWorkers)

	return neededIdleWorkers
}

// Serve starts serving
func (a *server) Serve() (Serving, error) {
	a.servingLock.Lock()
	defer a.servingLock.Unlock()

	if a.serving {
		return nil, ErrAlreadyServing
	}

	// Clear the old acceptorCloser signal
	<-a.acceptorCloser

	minWorkers := a.cfg.MinWorkers

	if minWorkers <= 0 {
		minWorkers = 1
	}

	// First, start listening
	listen, listenErr := a.listener.Listen()

	if listenErr != nil {
		return nil, listenErr
	}

	// Then, bring up worker producer
	a.workerProducerCloseWaiter.Add(1)

	go a.producer(minWorkers)

	// After that, bring up MinWorkers numbers of workers
	a.startWorkers(minWorkers)
	a.workerStarted += minWorkers

	// Then, start acceptor
	acceptors := uint32(1)

	if a.cfg.AcceptorPerWorkers > 0 {
		acceptors = a.cfg.MaxWorkers / a.cfg.AcceptorPerWorkers

		if acceptors <= 0 {
			acceptors = 1
		}
	}

	killChans := make([]chan kill, acceptors)

	for killChanIndx := range killChans {
		a.acceptorCloseWaiter.Add(1)

		killChans[killChanIndx] = make(chan kill, 1)

		go a.acceptor(uint64(killChanIndx), killChans[killChanIndx])
	}

	// Last, start server
	a.serverCloseWaiter.Add(1)

	go a.serve(listen)

	// Mark us as started
	a.serving = true

	return serving{
		Server:   a,
		Accepter: listen,
		Kills:    killChans,
	}, nil
}

// Listening returns listening port of current server
func (a serving) Listening() net.Addr {
	return a.Accepter.Addr()
}

// Close shuts down the server
func (a serving) Close() error {
	a.Server.servingLock.Lock()
	defer a.Server.servingLock.Unlock()

	if !a.Server.serving {
		return ErrAlreadyClosed
	}

	// First, shutdown the server so we don't accept connection
	// any more
	a.Server.serverCloser <- struct{}{}

	closeErr := a.Accepter.Close()

	if closeErr != nil {
		return closeErr
	}

	a.Server.serverCloseWaiter.Wait()
	<-a.Server.serverCloser

	// Producer require two shutdown signal to shutdown, first one
	// is notification for initialize it's shutdown progress so it will
	// stop trying to produce new workers when it detects the total worker
	// count became smaller than MinWorkers
	a.Server.workerProducerCloser <- struct{}{}

	// Send signal to kill exisiting connections, so the
	// worker will be released free to pick up shutdown signal
	fResults := make([]kill, len(a.Kills))

	for f := range a.Kills {
		fResults[f] = kill{
			Done: make(chan struct{}),
		}

		a.Kills[f] <- fResults[f]
	}

	for f := range fResults {
		<-fResults[f].Done
	}

	// First, shutdown workers
	a.Server.workerCloser <- struct{}{}
	a.Server.workerCloseWaiter.Wait()
	<-a.Server.workerCloser

	// Shutdown worker producer after all worker is down
	a.Server.workerProducerCloser <- struct{}{}
	a.Server.workerProducerCloseWaiter.Wait()

	// Then, shutdown acceptor
	a.Server.acceptorCloser <- struct{}{}
	a.Server.acceptorCloseWaiter.Wait()
	// Notice we don't clear acceptorCloser signal here. We did it
	// when initialize Serving so we can use that signal in `accept`
	// function even when the server is down

	a.Server.serving = false

	a.Server.logger.Debugf("Closed")

	return nil
}
