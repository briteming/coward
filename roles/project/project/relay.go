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

package project

import (
	"io"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/projector/request/join"
)

type relaying struct {
	dial       network.Dial
	projection Endpoint
	client     network.Connection
}

func (h *relaying) Initialize(l logger.Logger, server relay.Server) error {
	var conn network.Connection
	var dialErr error

	for cRetry := uint8(0); cRetry < h.projection.Retry; cRetry++ {
		conn, dialErr = h.dial.Dial()

		if dialErr == nil {
			break
		}
	}

	if dialErr != nil {
		rw.WriteFull(server, []byte{
			join.RespondClientRelayInitializationFailed})

		return dialErr
	}

	conn.SetTimeout(h.projection.Timeout)

	_, wErr := rw.WriteFull(
		server, []byte{join.RespondClientRelayInitialized})

	if wErr != nil {
		return wErr
	}

	h.client = conn

	return nil
}

func (h *relaying) Abort(l logger.Logger, aborter relay.Aborter) error {
	return aborter.Goodbye()
}

func (h *relaying) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	// This is "Server" (endpoint) -side of relay, no need to set
	// req timeout as the req connection will be shutdown once the
	// accessor goes away
	return h.client, nil
}
