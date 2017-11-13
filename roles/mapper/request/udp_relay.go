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

package request

import (
	"errors"
	"io"

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
	mapper proxycommon.MapID
	client io.ReadWriteCloser
}

func (c udpRelay) Initialize(server relay.Server) error {
	_, wErr := server.Write([]byte{request.UDPCommandTransport, byte(c.mapper)})

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
		return ErrUDPServerInitialUnknownError
	}

	// Send close, let the server knows that we'll go away
	server.SendSignal(relay.SignalCompleted, relay.SignalClose)

	return initError
}

func (c udpRelay) Client(server relay.Server) (io.ReadWriteCloser, error) {
	return c.client, nil
}
