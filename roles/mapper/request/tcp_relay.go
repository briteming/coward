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

package request

import (
	"errors"
	"io"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	proxycommon "github.com/reinit/coward/roles/proxy/common"
	"github.com/reinit/coward/roles/proxy/request"
)

// Errors
var (
	ErrTCPInitialRespondGeneralError = errors.New(
		"Failed to establish TCP Mapping due to a general error")

	ErrTCPInitialRespondAccessDeined = errors.New(
		"Failed to establish TCP Mapping as remote has deined the request")

	ErrTCPInitialRespondTargetUnreachable = errors.New(
		"Failed to establish TCP Mapping as remote has failed to " +
			"connect to the target")

	ErrTCPInitialRespondTargetUndefined = errors.New(
		"Failed to establish TCP Mapping as the target was undefined on " +
			"the remote")

	ErrTCPInitialRespondUnknownError = errors.New(
		"Failed to establish TCP Mapping due to unknown error")

	ErrTCPInitialRelayFailed = errors.New(
		"Remote Relay has failed to be initialized")
)

type tcpRelay struct {
	mapper  proxycommon.MapID
	client  network.Connection
	timeout time.Duration
}

func (c tcpRelay) Initialize(l logger.Logger, server relay.Server) error {
	_, wErr := rw.WriteFull(
		server, []byte{request.TCPCommandMapping, byte(c.mapper)})

	if wErr != nil {
		return wErr
	}

	command := [1]byte{}

	_, crErr := io.ReadFull(server, command[:])

	if crErr != nil {
		server.Done()

		return crErr
	}

	server.Done()

	var initError error

	switch command[0] {
	case request.TCPRespondOK:
		return nil

	case request.TCPRespondGeneralError:
		return ErrTCPInitialRespondGeneralError

	case request.TCPRespondMappingNotFound:
		return ErrTCPInitialRespondTargetUndefined

	case request.TCPRespondAccessDeined:
		initError = ErrTCPInitialRespondAccessDeined

	case request.TCPRespondUnreachable:
		initError = ErrTCPInitialRespondTargetUnreachable

	case byte(relay.SignalError):
		return ErrTCPInitialRelayFailed

	default:
		l.Debugf("Server responded with an unknown TCP initial result: %d",
			command[0])

		return ErrTCPInitialRespondUnknownError
	}

	// Send close, let the server knows that we'll go away
	server.Goodbye()

	return initError
}

func (c tcpRelay) Abort(l logger.Logger, aborter relay.Aborter) error {
	return aborter.Goodbye()
}

func (c tcpRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	return &relayConn{
		Connection:      c.client,
		timeout:         c.timeout,
		timeoutExpanded: false,
	}, nil
}
