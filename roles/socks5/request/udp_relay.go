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
	"net"
	"time"

	"github.com/reinit/coward/common/corunner"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/proxy/request"
	"github.com/reinit/coward/roles/socks5/common"
)

// Errors
var (
	ErrUDPInvalidLocalIP = errors.New(
		"Invalid Local IP address")

	ErrUDPInvalidClientIP = errors.New(
		"Invalid Client IP address")

	ErrUDPInvalidRequest = errors.New(
		"Invalid UDP request")

	ErrUDPServerFailedToListen = errors.New(
		"Remote has failed to initialize UDP listen")

	ErrUDPServerInvalidLocalAddr = errors.New(
		"The UDP listening address is invalid")

	ErrUDPServerRelayFailed = errors.New(
		"Remote Relay failed to be initialized")

	ErrUDPUnknownError = errors.New(
		"Unknown UDP error")
)

type udpRelay struct {
	client         network.Connection
	addr           common.Address
	requestTimeout time.Duration
	runner         corunner.Runner
	cancel         <-chan struct{}
	udpConn        io.ReadWriteCloser
	comfirmData    []byte
}

func (u *udpRelay) Initialize(server relay.Server) error {
	spServerHost, _, spServerErr := net.SplitHostPort(
		u.client.LocalAddr().String())

	if spServerErr != nil {
		return spServerErr
	}

	reqServerIP := net.ParseIP(spServerHost)

	if reqServerIP == nil {
		return ErrUDPInvalidLocalIP
	}

	spClientHost, _, spClientErr := net.SplitHostPort(
		u.client.RemoteAddr().String())

	if spClientErr != nil {
		return spClientErr
	}

	reqClientIP := net.ParseIP(spClientHost)

	if reqClientIP == nil {
		return ErrUDPInvalidClientIP
	}

	udpListener, udpListenErr := net.ListenUDP("udp", &net.UDPAddr{
		IP:   reqServerIP,
		Port: 0,
		Zone: "",
	})

	if udpListenErr != nil {
		return udpListenErr
	}

	runnerCloseResult, runErr := u.runner.Run(func() error {
		defer udpListener.Close()

		holdingBuf := [1]byte{}

		for {
			_, rErr := u.client.Read(holdingBuf[:])

			if rErr == nil {
				continue
			}

			return nil
		}
	}, u.cancel)

	if runErr != nil {
		return runErr
	}

	u.udpConn = &udpConn{
		UDPConn:            udpListener,
		allowedIP:          reqClientIP,
		accConn:            u.client,
		accConnCloseResult: runnerCloseResult,
		client:             nil,
	}

	// Now ask server to open a port for us
	_, wErr := server.Write([]byte{request.UDPCommandDelegate})

	if wErr != nil {
		u.udpConn.Close()

		return wErr
	}

	serverResp := [1]byte{}

	_, crErr := io.ReadFull(server, serverResp[:])

	if crErr != nil {
		server.Done()

		u.udpConn.Close()

		return crErr
	}

	server.Done()

	var serverRespErr error

	switch serverResp[0] {
	case request.UDPRespondOK:
		// Tell client that we're ready
		//
		// +----+-----+-------+------+----------+----------+
		// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
		// +----+-----+-------+------+----------+----------+
		// | 1  |  1  | X'00' |  1   | Variable |    2     |
		// +----+-----+-------+------+----------+----------+

		udpAddr := udpListener.LocalAddr().(*net.UDPAddr)
		udpPort := [2]byte{}
		udpPort[0] = byte(uint16(udpAddr.Port) >> 8)
		udpPort[1] = byte((uint16(udpAddr.Port) << 8) >> 8)

		if ipAddr := udpAddr.IP.To4(); ipAddr != nil {
			u.comfirmData = []byte{
				0x05, 0x00, 0x00, byte(common.ATypeIPv4),
				ipAddr[0], ipAddr[1], ipAddr[2], ipAddr[3],
				udpPort[0], udpPort[1]}

			return nil
		} else if ipAddr := udpAddr.IP.To16(); ipAddr != nil {
			u.comfirmData = []byte{
				0x05, 0x00, 0x00, byte(common.ATypeIPv6),
				ipAddr[0], ipAddr[1], ipAddr[2], ipAddr[3], ipAddr[4],
				ipAddr[5], ipAddr[6], ipAddr[7], ipAddr[8], ipAddr[9],
				ipAddr[10], ipAddr[11], ipAddr[12], ipAddr[13], ipAddr[14],
				ipAddr[15], udpPort[0], udpPort[1]}

			return nil
		}

		serverRespErr = ErrUDPServerInvalidLocalAddr

	case request.UDPRespondFailedToListen:
		serverRespErr = ErrUDPServerFailedToListen

	case request.UDPRespondInvalidRequest:
		u.udpConn.Close()

		return ErrUDPInvalidRequest

	case byte(relay.SignalError):
		u.udpConn.Close()

		return ErrUDPServerRelayFailed

	default:
		u.udpConn.Close()

		return ErrUDPUnknownError
	}

	u.udpConn.Close()

	server.SendSignal(relay.SignalCompleted, relay.SignalClose)

	return serverRespErr
}

func (u *udpRelay) Client(server relay.Server) (io.ReadWriteCloser, error) {
	_, wErr := rw.WriteFull(u.client, u.comfirmData)

	if wErr != nil {
		return nil, wErr
	}

	return u.udpConn, nil
}
