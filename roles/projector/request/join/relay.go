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

package join

import (
	"io"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
)

// requestRelay builds a request relay
type requestRelay struct {
	client  network.Connection
	timeout time.Duration
}

// relayClient wraps the Accessor connection
type relayClient struct {
	network.Connection

	timeout         time.Duration
	timeoutExpanded bool
}

// Read read the Accessor Conn and update it's timeout when needed
func (r *relayClient) Read(b []byte) (int, error) {
	rLen, rErr := r.Connection.Read(b)

	if rErr != nil || r.timeoutExpanded {
		return rLen, rErr
	}

	// We only expand the timeout when the client successfully sent
	// it's first message. Otherwise the client may aleady down without
	// us knowing it.
	r.Connection.SetTimeout(r.timeout)

	r.timeoutExpanded = true

	return rLen, rErr
}

// Initialize does nothing
func (r requestRelay) Initialize(l logger.Logger, server relay.Server) error {
	return nil
}

// Client returns the client
func (r requestRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	r.client.SetTimeout(r.timeout)

	return &relayClient{
		Connection:      r.client,
		timeout:         r.timeout,
		timeoutExpanded: false,
	}, nil
}
