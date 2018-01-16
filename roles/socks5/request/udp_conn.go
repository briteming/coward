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
	"math"
	"net"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/proxy/request"
	"github.com/reinit/coward/roles/socks5/common"
)

// Errors
var (
	ErrUDPConnUnauthorizedClient = errors.New(
		"UDP data received from an unauthorized client")

	ErrUDPConnInvalidAddressDataType = errors.New(
		"Invalid UDP address type")

	ErrUDPConnInvalidAddressLength = errors.New(
		"Invalid UDP address length")

	ErrUDPClientUnavailable = errors.New(
		"UDP client is unavailable")

	ErrUDPInvalidDataReceived = errors.New(
		"Invalid UDP data received")
)

type udpConn struct {
	request.UDPConn

	accConn            network.Connection
	accConnCloseResult chan error
	allowedIP          net.IP
	client             *net.UDPAddr
}

func (c *udpConn) Read(b []byte) (int, error) {
	for {
		rLen, rAddr, rErr := c.UDPConn.ReadFromUDP(b)

		if rErr != nil {
			return rLen, rErr
		}

		if !rAddr.IP.Equal(c.allowedIP) {
			continue
		}

		if c.client == nil {
			c.client = rAddr
		}

		// We also use rLen to check the buffer size
		if rLen <= 4 {
			continue
		}

		// UDP packet can only be read once, so we must parse it right
		// away.
		// The relay will give us 4k buffer, it's well enough for us
		// to parse the UDP headers and convert them to proxy command.
		// But remember this pre-condition. Once it changed, following
		// code may not working and cause panic.

		// We don't support FRAG.
		if b[2] != 0x00 {
			continue
		}

		targetAddr := common.Address{}

		addrReadLen, addrReadErr := targetAddr.ReadFrom(
			rw.ByteSliceReader(b[3:]))

		if addrReadErr != nil {
			return 0, addrReadErr
		}

		switch targetAddr.AType {
		case common.ATypeIPv4:
			if rLen <= 10 || addrReadLen != 7 {
				continue
			}

			b[0] = request.UDPSendIPv4

			copy(b[1:5], targetAddr.Address[0:4])

			b[5] = byte(targetAddr.Port >> 8)
			b[6] = byte((targetAddr.Port << 8) >> 8)

			return copy(b[7:], b[10:7+rLen]), nil

		case common.ATypeIPv6:
			if rLen <= 22 || addrReadLen != 19 {
				continue
			}

			b[0] = request.UDPSendIPv6

			copy(b[1:17], targetAddr.Address[0:16])

			b[17] = byte(targetAddr.Port >> 8)
			b[18] = byte((targetAddr.Port << 8) >> 8)

			return copy(b[19:], b[22:19+rLen]), nil

		case common.ATypeHost:
			addrLen := len(targetAddr.Address)

			if addrLen > math.MaxUint8 {
				return 0, ErrUDPConnInvalidAddressLength
			}

			if rLen <= addrLen+6 {
				continue
			}

			b[0] = request.UDPSendHost
			b[1] = byte(addrLen)

			copy(b[2:2+addrLen], targetAddr.Address[0:addrLen])

			b[addrLen+2] = byte(targetAddr.Port >> 8)
			b[addrLen+3] = byte((targetAddr.Port << 8) >> 8)

			return copy(b[addrLen+3:], b[6+addrLen:addrLen+3+rLen]), nil

		default:
			return 0, ErrUDPConnInvalidAddressDataType
		}
	}
}

// Write the data from Transceiver server to socks5 UDP client
func (c *udpConn) Write(b []byte) (int, error) {
	if c.client == nil {
		return 0, ErrUDPClientUnavailable
	}

	bLen := len(b)

	if bLen < 1 {
		return 0, ErrUDPInvalidDataReceived
	}

	// Transceiver's feedback data is smaller than Socks5 UDP request
	// header by 3 bytes (Don't have RSV && FRAG).
	writeBuf := make([]byte, bLen+3)

	switch b[0] {
	case request.UDPSendIPv4:
		writeBuf[3] = byte(common.ATypeIPv4)

	case request.UDPSendIPv6:
		writeBuf[3] = byte(common.ATypeIPv6)

	case request.UDPSendHost:
		writeBuf[3] = byte(common.ATypeHost)

	default:
		return 0, ErrUDPConnInvalidAddressDataType
	}

	copy(writeBuf[4:], b[1:bLen])

	_, wErr := c.UDPConn.WriteToUDP(writeBuf, c.client)

	return bLen, wErr
}

func (c *udpConn) Close() error {
	c.accConn.Close()

	<-c.accConnCloseResult

	return c.UDPConn.Close()
}
