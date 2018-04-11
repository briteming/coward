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
	"math"
	"time"

	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/proxy/request"
	"github.com/reinit/coward/roles/socks5/common"
)

// Errors
var (
	ErrInvalidAddressData = errors.New(
		"Invalid Address data")
)

type connectRelay struct {
	client         io.ReadWriteCloser
	addr           common.Address
	requestTimeout time.Duration
}

func (c connectRelay) Initialize(l logger.Logger, server relay.Server) error {
	var wErr error

	// Initialize the Channel to Connect command
	// +-----+------+------+------------+
	// | CMD | Addr | Port | ReqTimeout |
	// +-----+------+------+------------+
	// |  1  |  N   |   2  |      1     |
	// +-----+------+------+------------+

	portTimeoutBytes := [3]byte{}

	portTimeoutBytes[0] = byte(c.addr.Port >> 8)
	portTimeoutBytes[1] = byte((c.addr.Port << 8) >> 8)

	reqTimeout := (c.requestTimeout / 2).Seconds()

	if reqTimeout > math.MaxUint8 {
		reqTimeout = math.MaxUint8
	} else if reqTimeout < 1 {
		reqTimeout = 1
	}

	portTimeoutBytes[2] = byte(reqTimeout)

	switch c.addr.AType {
	case common.ATypeIPv4:
		_, wErr = rw.WriteFull(server, []byte{
			request.TCPCommandIPv4,
			c.addr.Address[0], c.addr.Address[1],
			c.addr.Address[2], c.addr.Address[3],
			portTimeoutBytes[0], portTimeoutBytes[1], portTimeoutBytes[2],
		})

	case common.ATypeIPv6:
		_, wErr = rw.WriteFull(server, []byte{
			request.TCPCommandIPv6,
			c.addr.Address[0], c.addr.Address[1],
			c.addr.Address[2], c.addr.Address[3],
			c.addr.Address[4], c.addr.Address[5],
			c.addr.Address[6], c.addr.Address[7],
			c.addr.Address[8], c.addr.Address[9],
			c.addr.Address[10], c.addr.Address[11],
			c.addr.Address[12], c.addr.Address[13],
			c.addr.Address[14], c.addr.Address[15],
			portTimeoutBytes[0], portTimeoutBytes[1], portTimeoutBytes[2],
		})

	case common.ATypeHost:
		addrLen := len(c.addr.Address)

		if addrLen > math.MaxUint8 {
			return ErrInvalidAddressData
		}

		hostData := make([]byte, addrLen+5)

		copy(hostData[2:2+addrLen], c.addr.Address)

		hostData[0] = request.TCPCommandHost
		hostData[1] = byte(addrLen)
		hostData[addrLen+2] = portTimeoutBytes[0]
		hostData[addrLen+3] = portTimeoutBytes[1]
		hostData[addrLen+4] = portTimeoutBytes[2]

		_, wErr = rw.WriteFull(server, hostData)

	default:
		return ErrConnectInvalidAddressType
	}

	if wErr != nil {
		return wErr
	}

	// Respond Format
	// +------+------+
	// | RESP | Data |
	// +------+------+
	// |  1   |      |
	// +------+------+
	command := [1]byte{}

	_, crErr := io.ReadFull(server, command[:])

	if crErr != nil {
		server.Done()

		return crErr
	}

	server.Done()

	var connectError error

	switch command[0] {
	case request.TCPRespondOK:
		return nil

	case request.TCPRespondGeneralError:
		return ErrConnectInitialRespondGeneralError

	case request.TCPRespondAccessDeined:
		connectError = ErrConnectInitialRespondAccessDeined

	case request.TCPRespondUnreachable:
		connectError = ErrConnectInitialRespondTargetUnreachable

	case request.TCPRespondBadRequest:
		return ErrConnectInitialFailedBadRequest

	case byte(relay.SignalError):
		return ErrConnectInitialRelayFailed

	default:
		l.Debugf("Server responded with an unknown TCP initial result code: %d",
			command[0])

		return ErrConnectInitialRespondUnknownError
	}

	// Send close, let the server knows that we'll go away
	server.Goodbye()

	return connectError
}

func (c connectRelay) Abort(l logger.Logger, aborter relay.Aborter) error {
	return aborter.Goodbye()
}

func (c connectRelay) Client(
	l logger.Logger, server relay.Server) (io.ReadWriteCloser, error) {
	// Tell client that we're ready
	//
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	_, wErr := rw.WriteFull(c.client, []byte{
		0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	if wErr != nil {
		return nil, wErr
	}

	return c.client, nil
}
