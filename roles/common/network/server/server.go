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
	"net"
	"sync"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
)

// Errors
var (
	ErrAlreadyServing = errors.New(
		"Already serving")

	ErrNotServing = errors.New(
		"Not serving")
)

// client registeration data
type client struct {
	Connection network.Connection
	Result     chan error
}

// leave connection unregisteration request
type leave struct {
	ID         string
	Connection network.Connection
}

// server implements network.Server
type server struct {
	listener   network.Listener
	handler    network.Handler
	logger     logger.Logger
	runner     worker.Runner
	cfg        Config
	accept     chan network.Connection
	leave      chan leave
	serving    bool
	downLock   sync.Mutex
	downWait   sync.WaitGroup
	downNotify chan struct{}
}

type serving struct {
	accepter network.Acceptor
	server   *server
}

// New creates a new network.Server
func New(
	listener network.Listener,
	handler network.Handler,
	logger logger.Logger,
	runner worker.Runner,
	cfg Config,
) network.Server {
	return &server{
		listener:   listener,
		handler:    handler,
		logger:     logger.Context("Server (" + listener.String() + ")"),
		runner:     runner,
		cfg:        cfg,
		accept:     make(chan network.Connection),
		leave:      make(chan leave, cfg.MaxConnections),
		serving:    false,
		downLock:   sync.Mutex{},
		downNotify: make(chan struct{}, 1),
		downWait:   sync.WaitGroup{},
	}
}

// acceptor accepts connection and send them to handler
func (s *server) acceptor() {
	log := s.logger.Context("Acceptor")

	defer func() {
		log.Debugf("Closed")

		s.downWait.Done()
	}()

	downNotify := s.downNotify
	acceptor := s.accept
	clients := make(map[string]client, s.cfg.MaxConnections)
	closing := false
	currentClients := uint64(0)
	maxClients := uint64(s.cfg.MaxConnections)

	for {
		select {
		case <-downNotify:
			downNotify <- struct{}{}

			downNotify = nil
			acceptor = nil
			closing = true

			if len(clients) <= 0 {
				return
			}

			for k := range clients {
				select {
				case <-clients[k].Connection.Closed():
				default:
					clients[k].Connection.Close()
				}
			}

			log.Debugf("Closing all clients")

		case cl := <-acceptor:
			if currentClients >= maxClients {
				cl.Close()

				log.Debugf("Failed to handle client \"%s\" because server "+
					"has reached it's capacity", cl.RemoteAddr())

				continue
			}

			connectionID := string(cl.ID()) + ":" + time.Now().String()

			_, cliFound := clients[connectionID]

			if cliFound {
				cl.Close()

				log.Debugf("Client \"%s\" already registered", cl.RemoteAddr())

				continue
			}

			h := handle{
				ID:         connectionID,
				Connection: cl,
				Leave:      s.leave,
				Handler:    s.handler,
			}

			runResult, runJoinErr := s.runner.Run(
				s.logger.Context("Client ("+cl.RemoteAddr().String()+")"),
				h.Handle,
				cl.Closed())

			if runJoinErr != nil {
				cl.Close()

				log.Debugf("Failed to handle client \"%s\" due to error: %s",
					cl.RemoteAddr(), runJoinErr)

				continue
			}

			currentClients++

			clients[connectionID] = client{
				Connection: cl,
				Result:     runResult,
			}

			log.Debugf("New client \"%s\"", cl.RemoteAddr())

		case cl := <-s.leave:
			cli, cliFound := clients[cl.ID]

			if !cliFound {
				panic("Removing an non-existing client from clients " +
					"record is not allowed")
			}

			currentClients--

			delete(clients, cl.ID)

			select {
			case <-cl.Connection.Closed():
			default:
				cl.Connection.Close()
			}

			select {
			case clErr := <-cli.Result:
				if clErr != nil {
					log.Debugf("Client \"%s\" is disconnected due to error: %s",
						cl.Connection.RemoteAddr(), clErr)
				}

			default:
			}

			log.Debugf("Client \"%s\" is disconnected",
				cl.Connection.RemoteAddr())

			if !closing {
				continue
			}

			if len(clients) > 0 {
				continue
			}

			return
		}
	}
}

// serve listens the accepter and send accepted connection to acceptor
func (s *server) serve(acc network.Acceptor) error {
	log := s.logger.Context("Serving")

	defer func() {
		log.Debugf("Closed")

		s.downWait.Done()
	}()

	for {
		cli, accErr := acc.Accept()

		if accErr != nil {
			select {
			case <-acc.Closed():
				return accErr

			default:
				log.Warningf("Failed to accept incomming connection "+
					"due to error: %s", accErr)

				time.Sleep(s.cfg.AcceptErrorWait)

				continue
			}
		}

		select {
		case <-s.downNotify:
			s.downNotify <- struct{}{}

		case s.accept <- cli:
		}
	}
}

// Serve starts serving
func (s *server) Serve() (network.Serving, error) {
	s.downLock.Lock()
	defer s.downLock.Unlock()

	if s.serving {
		return nil, ErrAlreadyServing
	}

	acc, listenErr := s.listener.Listen()

	if listenErr != nil {
		return nil, listenErr
	}

	s.downWait.Add(2)

	go s.acceptor()
	go s.serve(acc)

	s.serving = true

	return serving{
		accepter: acc,
		server:   s,
	}, nil
}

// Listening returns local address that current server is listening to
func (s serving) Listening() net.Addr {
	return s.accepter.Addr()
}

// Close shutdown current server
func (s serving) Close() error {
	s.server.downLock.Lock()
	defer s.server.downLock.Unlock()

	if !s.server.serving {
		return ErrNotServing
	}

	s.server.downNotify <- struct{}{}

	defer func() {
		<-s.server.downNotify
	}()

	cErr := s.accepter.Close()

	if cErr != nil {
		return cErr
	}

	s.server.downWait.Wait()

	s.server.serving = false

	return nil
}
