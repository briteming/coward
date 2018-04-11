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
	ErrUDPServerInitialTargetUndefined = errors.New(
		"Remote has failed to initialize UDP mapping request as the target " +
			"was undefined on the remote")

	ErrUDPServerInitialTargetUnresolved = errors.New(
		"Remote has failed to initialize UDP mapping request as remote " +
			"failed to resolve the target address")

	ErrUDPServerInitialRemoteFailedToListen = errors.New(
		"Remote has failed to initialize UDP mapping request as remote " +
			"failed to establish listen on the ephemeral port")

	ErrUDPServerInitialInvalidRequest = errors.New(
		"Remote has failed to initialize UDP mapping request as remote " +
			"failed to handle the request")

	ErrUDPServerInitialUnknownError = errors.New(
		"Remote has failed to initialize UDP mapping request due to an " +
			"unknown error")

	ErrUDPServerRelayFailed = errors.New(
		"Remote Relay initialization has failed")
)

type udpRelay struct {
	mapper  proxycommon.MapID
	client  network.Connection
	timeout time.Duration
}

func (c udpRelay) Initialize(l logger.Logger, server relay.Server) error {
	_, wErr := rw.WriteFull(
		server, []byte{request.UDPCommandTransport, byte(c.mapper)})

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
	case request.UDPRespondOK:
		return nil

	case request.UDPRespondMappingNotFound:
		return ErrUDPServerInitialTargetUndefined

	case request.UDPRespondMappingHostUnresolved:
		initError = ErrUDPServerInitialTargetUnresolved

	case request.UDPRespondFailedToListen:
		initError = ErrUDPServerInitialRemoteFailedToListen

	case request.UDPRespondInvalidRequest:
		return ErrUDPServerInitialInvalidRequest

	case byte(relay.SignalError):
		return ErrUDPServerRelayFailed

	default:
		l.Debugf("Server responded with an unknown UDP initial result: %d",
			command[0])

		return ErrUDPServerInitialUnknownError
	}

	// Send close, let the server knows that we'll go away
	server.Goodbye()

	return initError
}

func (c udpRelay) Abort(l logger.Logger, aborter relay.Aborter) error {
	return aborter.Goodbye()
}

func (c udpRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	return &relayConn{
		Connection:      c.client,
		timeout:         c.timeout,
		timeoutExpanded: false,
	}, nil
}
