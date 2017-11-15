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
	"io"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
)

// requestRelay builds a request relay
type requestRelay struct {
	client network.Connection
}

// Initialize does nothing
func (r requestRelay) Initialize(l logger.Logger, server relay.Server) error {
	return nil
}

// Client returns the client
func (r requestRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	return r.client, nil
}
