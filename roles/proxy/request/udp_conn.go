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
	"math"
	"net"
	"sync"

	"github.com/reinit/coward/roles/common/network/resolve"
)

// Errors
var (
	ErrUDPTransportTooManyDestinations = errors.New(
		"Too many UDP destinations")

	ErrUDPTransportWritingInvalidData = errors.New(
		"Writing invalid UDP data")

	ErrUDPTransportLocalAccessDeined = errors.New(
		"Local UDP access deined")
)

// UDPConn is the Conn of UDP
type UDPConn interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	Close() error
}

type udpConn struct {
	UDPConn

	resolver   resolve.Resolver
	remotes    map[resolve.IPMark]struct{}
	maxRemotes uint32
	remoteLock sync.RWMutex
}

// Read reads data from a UDP remote, convert it and it send it back
// to the Transceiver Client.
// Notice any data that larger than len(b) will simply be dropped
func (u *udpConn) Read(b []byte) (int, error) {
	bLen := len(b)
	rBuf := make([]byte, bLen)

	for {
		rLen, rAddr, rErr := u.UDPConn.ReadFromUDP(rBuf)

		if rErr != nil {
			return 0, rErr
		}

		addressSendType := UDPSendHost

		knownHost, reverseErr := u.resolver.Reverse(rAddr.IP)

		if reverseErr != nil {
			ipMark := resolve.IPMark{}
			ipMark.Import(rAddr.IP)

			u.remoteLock.RLock()
			_, found := u.remotes[ipMark]
			u.remoteLock.RUnlock()

			if !found {
				continue
			}

			if knownHost = string(rAddr.IP.To4()); knownHost != "" {
				addressSendType = UDPSendIPv4
			} else if knownHost = string(rAddr.IP.To16()); knownHost != "" {
				addressSendType = UDPSendIPv6
			} else {
				continue
			}
		}

		portBytes := [2]byte{}
		portBytes[0] |= byte(rAddr.Port >> 8)
		portBytes[1] |= byte((rAddr.Port << 8) >> 8)

		switch addressSendType {
		case UDPSendIPv4:
			if bLen < 7 {
				continue
			}

			headBuf := [7]byte{}

			headBuf[0] = UDPSendIPv4

			copy(headBuf[1:5], knownHost[0:4])

			headBuf[5] = portBytes[0]
			headBuf[6] = portBytes[1]

			parsedLen := copy(b, headBuf[:])
			parsedLen += copy(b[7:], rBuf[:rLen])

			return parsedLen, nil

		case UDPSendIPv6:
			if bLen < 19 {
				continue
			}

			headBuf := [19]byte{}

			headBuf[0] = UDPSendIPv6

			copy(headBuf[1:17], knownHost[0:16])

			headBuf[17] = portBytes[0]
			headBuf[18] = portBytes[1]

			parsedLen := copy(b, headBuf[:])
			parsedLen += copy(b[19:], rBuf[:rLen])

			return parsedLen, nil

		case UDPSendHost:
			hostLen := len(knownHost)

			if hostLen > int(math.MaxUint8) || hostLen+4 > bLen {
				continue
			}

			headBuf := make([]byte, hostLen+4)

			headBuf[0] = UDPSendHost
			headBuf[1] = byte(hostLen)

			copy(headBuf[2:hostLen+2], knownHost[0:hostLen])

			headBuf[hostLen+2] = portBytes[0]
			headBuf[hostLen+3] = portBytes[1]

			parsedLen := copy(b, headBuf[:])
			parsedLen += copy(b[hostLen+4:], rBuf[:rLen])

			return parsedLen, nil

		default:
			continue
		}
	}
}

func (u *udpConn) recordDestination(ipM resolve.IPMark) error {
	u.remoteLock.RLock()

	_, found := u.remotes[ipM]

	if !found && uint32(len(u.remotes)) >= u.maxRemotes {
		u.remoteLock.RUnlock()

		return ErrUDPTransportTooManyDestinations
	}

	u.remoteLock.RUnlock()

	u.remoteLock.Lock()
	u.remotes[ipM] = struct{}{}
	u.remoteLock.Unlock()

	return nil
}

// Write convert data that transmitted from a client and send it
// to the specified UDP remote
func (u *udpConn) Write(b []byte) (int, error) {
	bLen := len(b)

	if bLen <= 1 {
		return 0, ErrUDPTransportWritingInvalidData
	}

	switch b[0] {
	case UDPSendIPv4:
		if bLen < 7 {
			return 0, ErrUDPTransportInvalidAddressData
		}

		ip := net.IPv4(b[1], b[2], b[3], b[4])

		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
			return 0, ErrUDPTransportLocalAccessDeined
		}

		port := uint16(0)
		port |= uint16(b[5])
		port <<= 8
		port |= uint16(b[6])

		ipM := resolve.IPMark{}
		ipM.Import(ip)

		rdErr := u.recordDestination(ipM)

		if rdErr != nil {
			return 0, rdErr
		}

		_, wErr := u.UDPConn.WriteToUDP(b[7:], &net.UDPAddr{
			IP:   ip,
			Port: int(port),
			Zone: "",
		})

		return bLen, wErr

	case UDPSendIPv6:
		if bLen < 19 {
			return 0, ErrUDPTransportInvalidAddressData
		}

		ip := net.IP{
			b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8],
			b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16],
		}

		if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
			return 0, ErrUDPTransportLocalAccessDeined
		}

		port := uint16(0)
		port |= uint16(b[17])
		port <<= 8
		port |= uint16(b[18])

		ipM := resolve.IPMark{}
		ipM.Import(ip)

		rdErr := u.recordDestination(ipM)

		if rdErr != nil {
			return 0, rdErr
		}

		_, wErr := u.UDPConn.WriteToUDP(b[19:], &net.UDPAddr{
			IP:   ip,
			Port: int(port),
			Zone: "",
		})

		return bLen, wErr

	case UDPSendHost:
		if bLen < int(b[1]+3) { // 1 Size, 2 Port
			return 0, ErrUDPTransportInvalidAddressData
		}

		hostName := string(b[2 : 2+b[1]])
		port := uint16(0)
		port |= uint16(b[b[1]+2])
		port <<= 8
		port |= uint16(b[b[1]+3])

		resolved, resolveErr := u.resolver.Resolve(hostName)

		if resolveErr != nil {
			return 0, resolveErr
		}

		if resolved[0].IsLoopback() ||
			resolved[0].IsUnspecified() ||
			resolved[0].IsMulticast() {
			return 0, ErrUDPTransportLocalAccessDeined
		}

		ipM := resolve.IPMark{}
		ipM.Import(resolved[0])

		rdErr := u.recordDestination(ipM)

		if rdErr != nil {
			return 0, rdErr
		}

		_, wErr := u.UDPConn.WriteToUDP(b[b[1]+4:], &net.UDPAddr{
			IP:   resolved[0],
			Port: int(port),
			Zone: "",
		})

		return bLen, wErr

	default:
		return 0, ErrUDPTransportUnknownAddressType
	}
}

func (u *udpConn) Close() error {
	return u.UDPConn.Close()
}
