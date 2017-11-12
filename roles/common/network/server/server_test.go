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
	"io"
	"net"
	"testing"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/roles/common/network"
)

type dummyIncoming struct{}

func (d *dummyIncoming) New(
	conn network.Connection,
	log logger.Logger,
) (Client, error) {
	return nil, nil
}

type dummyListener struct{}
type dummyAcceptor struct {
	acceptErrChan chan error
}

func (d *dummyListener) Listen() (network.Acceptor, error) {
	return &dummyAcceptor{
		acceptErrChan: make(chan error),
	}, nil
}

func (d *dummyListener) String() string {
	return "LISTENER"
}

func (d *dummyAcceptor) Addr() net.Addr {
	return nil
}

func (d *dummyAcceptor) Accept() (network.Connection, error) {
	return nil, <-d.acceptErrChan
}

func (d *dummyAcceptor) Close() error {
	d.acceptErrChan <- io.EOF

	return nil
}

func TestServerUpDown(t *testing.T) {
	s := New(&dummyListener{}, &dummyIncoming{}, logger.NewDitch(), Config{
		MaxWorkers:         1024,
		MinWorkers:         32,
		MaxWorkerIdle:      10 * time.Second,
		AcceptErrorWait:    1 * time.Second,
		AcceptorPerWorkers: 1000,
	})

	for i := 0; i < 100; i++ {
		serve, serveErr := s.Serve()

		if serveErr != nil {
			t.Error("Failed to serve due to error:", serveErr)

			return
		}

		closeErr := serve.Close()

		if closeErr != nil {
			t.Error("Failed to close due to error:", closeErr)

			return
		}
	}

	serve, serveErr := s.Serve()

	if serveErr != nil {
		t.Error("Failed to serve due to error:", serveErr)

		return
	}

	closeErr := serve.Close()

	if closeErr != nil {
		t.Error("Failed to close due to error:", closeErr)

		return
	}

	closeErr = serve.Close()

	if closeErr == nil {
		t.Error("Close a closed server must resulting an error")

		return
	}

	serve, serveErr = s.Serve()

	if serveErr != nil {
		t.Error("Failed to serve due to error:", serveErr)

		return
	}

	_, serveErr = s.Serve()

	if serveErr == nil {
		t.Error("Start a already serving server must resulting an error")

		return
	}

	closeErr = serve.Close()

	if closeErr != nil {
		t.Error("Failed to close due to error:", closeErr)

		return
	}
}
