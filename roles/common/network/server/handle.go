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
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/roles/common/network"
)

// handle handles server client, and notify it's drop
type handle struct {
	ID         string
	Connection network.Connection
	Leave      chan leave
	Handler    network.Handler
}

// Handle start handle the client
func (h handle) Handle(log logger.Logger) error {
	defer func() {
		h.Leave <- leave{
			ID:         h.ID,
			Connection: h.Connection,
		}
	}()

	c, cErr := h.Handler.New(h.Connection, log)

	if cErr != nil {
		return cErr
	}

	return c.Serve()
}
