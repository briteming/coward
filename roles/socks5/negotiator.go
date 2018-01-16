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

package socks5

import (
	"errors"
	"io"

	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/socks5/common"
	"github.com/reinit/coward/roles/socks5/request"
)

// Errors
var (
	ErrNegoUnsupportedSocksVersion = errors.New(
		"Unsupported Socks version")

	ErrNegoUnsupportedSocksAuthNegoVersion = errors.New(
		"Unsupported Socks authentication subnegotiation version")

	ErrNegoNMETHODsMustBeClarified = errors.New(
		"Socks5 NMETHOD must be clarified")

	ErrNegoUnsupportedNMETHOD = errors.New(
		"Unsupported Socks5 NMETHOD")

	ErrNegoAuthFailedToReadHead = errors.New(
		"Failed to read head data for Socks5 user authentication")

	ErrNegoAuthFailedToReadUsername = errors.New(
		"Failed to read username for Socks5 user authentication")

	ErrNegoAuthFailedToReadPassword = errors.New(
		"Failed to read password for Socks5 user authentication")

	ErrNegoRequestFailedToReadHead = errors.New(
		"Failed to read the header of Socks5 request")

	ErrNegoUnsupportedCommand = errors.New(
		"Unsupported Socks5 command")

	ErrNegoUnsupportedAType = errors.New(
		"Unsupported Socks5 address type")
)

type nmethod byte
type cmd byte

const (
	nmethodNoAuth      nmethod = 0x00
	nmethodGSSAPI      nmethod = 0x01
	nmethodUserName    nmethod = 0x02
	nmethodUnsupported nmethod = 0xff
)

const (
	cmdConnect cmd = 0x01
	cmdBind    cmd = 0x02
	cmdUDP     cmd = 0x03
)

type negotiator struct {
	cfg                    Config
	conn                   network.Connection
	runner                 worker.Runner
	shb                    *common.SharedBuffers
	authenticator          Authenticator
	selectedCMD            cmd
	selectedAddress        common.Address
	selectedRequestBuilder transceiver.RequestBuilder
}

func (n *negotiator) Bootup() (fsm.State, error) {
	headerBuf := [2]byte{}

	_, rErr := io.ReadFull(n.conn, headerBuf[:])

	if rErr != nil {
		return nil, rErr
	}

	if headerBuf[0] != 0x05 {
		return nil, ErrNegoUnsupportedSocksVersion
	}

	if headerBuf[1] <= 0 {
		return nil, ErrNegoNMETHODsMustBeClarified
	}

	nmethodsBuf := [256]byte{}

	_, rErr = io.ReadFull(n.conn, nmethodsBuf[:headerBuf[1]])

	if rErr != nil {
		return nil, rErr
	}

	if n.authenticator != nil {
		for _, v := range nmethodsBuf[:headerBuf[1]] {
			if nmethod(v) != nmethodUserName {
				continue
			}

			_, wErr := rw.WriteFull(n.conn, []byte{0x05, byte(nmethodUserName)})

			if wErr != nil {
				return nil, wErr
			}

			return n.auth, nil
		}
	} else {
		for _, v := range nmethodsBuf[:headerBuf[1]] {
			if nmethod(v) != nmethodNoAuth {
				continue
			}

			_, wErr := rw.WriteFull(n.conn, []byte{0x05, byte(nmethodNoAuth)})

			if wErr != nil {
				return nil, wErr
			}

			return n.request, nil
		}
	}

	rw.WriteFull(n.conn, []byte{0x05, byte(nmethodUnsupported)})

	return nil, ErrNegoUnsupportedNMETHOD
}

func (n *negotiator) Shutdown() error {
	return nil
}

func (n *negotiator) Build() (
	transceiver.Destination,
	transceiver.BalancedRequestBuilder,
	error,
) {
	switch n.selectedCMD {
	case cmdConnect:
		return "Connect:" + transceiver.Destination(
				n.selectedAddress.Address,
			), request.Connect(
				n.conn,
				n.selectedAddress,
				n.runner,
				n.shb,
				n.cfg.NegotiationTimeout), nil

	case cmdUDP:
		return "UDP:" + transceiver.Destination(
				n.selectedAddress.Address,
			), request.UDP(
				n.conn,
				n.selectedAddress,
				n.runner,
				n.shb,
				n.cfg.NegotiationTimeout), nil

	default:
		// +----+-----+-------+------+----------+----------+
		// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
		// +----+-----+-------+------+----------+----------+
		// | 1  |  1  | X'00' |  1   | Variable |    2     |
		// +----+-----+-------+------+----------+----------+
		rw.WriteFull(n.conn, []byte{
			0x05, 0x07, 0x00,
			0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

		return "", nil, ErrNegoUnsupportedCommand
	}
}

func (n *negotiator) auth(f fsm.FSM) error {
	// See https://tools.ietf.org/html/rfc1929
	var rLen int

	readBuf := [256]byte{}

	_, rErr := io.ReadFull(n.conn, readBuf[:2])

	if rErr != nil {
		return ErrNegoAuthFailedToReadHead
	}

	if readBuf[0] != 0x01 {
		rw.WriteFull(n.conn, []byte{0x05, 0x01})

		return ErrNegoUnsupportedSocksAuthNegoVersion
	}

	// Read user name
	if readBuf[1] <= 0 {
		return ErrNegoAuthFailedToReadUsername
	}

	rLen, rErr = io.ReadFull(n.conn, readBuf[:readBuf[1]])

	if rErr != nil {
		return ErrNegoAuthFailedToReadUsername
	}

	userName := make([]byte, rLen)
	copy(userName, readBuf[:rLen])

	// Read password
	_, rErr = io.ReadFull(n.conn, readBuf[:1])

	if rErr != nil {
		return ErrNegoAuthFailedToReadPassword
	}

	if readBuf[0] <= 0 {
		// No password? Good, auth now only with the username
		aErr := n.authenticator(string(userName), "")

		if aErr != nil {
			rw.WriteFull(n.conn, []byte{0x05, 0x01})

			return aErr
		}

		_, wErr := rw.WriteFull(n.conn, []byte{0x05, 0x00})

		if wErr != nil {
			return wErr
		}

		f.Switch(n.request)

		return nil
	}

	rLen, rErr = io.ReadFull(n.conn, readBuf[:readBuf[0]])

	if rErr != nil {
		return ErrNegoAuthFailedToReadPassword
	}

	aErr := n.authenticator(string(userName), string(readBuf[:rLen]))

	if aErr != nil {
		rw.WriteFull(n.conn, []byte{0x05, 0x01})

		return aErr
	}

	_, wErr := rw.WriteFull(n.conn, []byte{0x05, 0x00})

	if wErr != nil {
		return wErr
	}

	f.Switch(n.request)

	return nil
}

func (n *negotiator) request(f fsm.FSM) error {
	// See https://www.ietf.org/rfc/rfc1928.txt
	//
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	readBuf := [3]byte{}

	_, rErr := io.ReadFull(n.conn, readBuf[:])

	if rErr != nil {
		return ErrNegoRequestFailedToReadHead
	}

	if readBuf[0] != 0x05 {
		return ErrNegoUnsupportedSocksVersion
	}

	n.selectedCMD = cmd(readBuf[1])

	_, rErr = n.selectedAddress.ReadFrom(n.conn)

	if rErr != nil {
		return rErr
	}

	return f.Shutdown()
}
