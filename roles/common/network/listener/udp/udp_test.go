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

package udp

import (
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/network/connection/tcp"
	udpdial "github.com/reinit/coward/roles/common/network/dialer/udp"
)

func TestUDPListenUpDown(t *testing.T) {
	buf := make([]byte, 1024)
	uu := New(net.ParseIP("127.0.0.1"), 0, 1*time.Second, 16, buf, tcp.Wrap)

	cc, lErr := uu.Listen()

	if lErr != nil {
		t.Error("Failed to listen due to error:", lErr)

		return
	}

	closeErr := cc.Close()

	if closeErr != nil {
		t.Error("Failed to close due to error:", closeErr)

		return
	}
}

func TestUDPListen(t *testing.T) {
	buf := make([]byte, 1024)
	uu := New(net.ParseIP("127.0.0.1"), 0, 1*time.Second, 16, buf, tcp.Wrap)

	acc, lErr := uu.Listen()

	if lErr != nil {
		t.Error("Failed to listen due to error:", lErr)

		return
	}

	defer acc.Close()

	addr := acc.Addr()

	spHost, spPort, spErr := net.SplitHostPort(addr.String())

	if spErr != nil {
		t.Error("Failed to split listening host port:", spErr)

		return
	}

	portN, portNErr := strconv.ParseUint(spPort, 10, 16)

	if portNErr != nil {
		t.Error("Failed to convert port string to number:", portNErr)

		return
	}

	const testCount = 2

	downWaitGroup := sync.WaitGroup{}
	sendStartGroup := sync.WaitGroup{}

	sendStartGroup.Add(1)
	downWaitGroup.Add(1)

	go func() {
		defer downWaitGroup.Done()

		sendStartGroup.Done()

		received := 0

		for {
			if received >= testCount {
				break
			}

			received++

			conn, accErr := acc.Accept()

			if accErr != nil {
				t.Error("Failed to accept:", accErr)

				return
			}

			downWaitGroup.Add(1)

			go func(c network.Connection) {
				defer func() {
					downWaitGroup.Done()

					c.Close()
				}()

				buf := [4092]byte{}

				c.SetTimeout(10 * time.Second)

				for {
					rLen, rErr := c.Read(buf[:])

					if rErr != nil {
						t.Error("Failed to read UDP data:", rErr)

						return
					}

					_, wErr := c.Write(buf[:rLen])

					if wErr != nil {
						t.Error("Failed to send UDP data:", wErr)

						return
					}

					return
				}
			}(conn)
		}
	}()

	type result struct {
		Data [4096]byte
		Len  int
	}

	resultData := [testCount]result{}

	for i := byte(0); i < testCount; i++ {
		downWaitGroup.Add(1)

		go func(id byte) {
			defer downWaitGroup.Done()

			sendStartGroup.Wait()

			dler := udpdial.New(spHost, uint16(portN), 10*time.Second, tcp.Wrap)
			dl, dlErr := dler.Dialer().Dial()

			if dlErr != nil {
				return
			}

			defer dl.Close()

			dl.SetTimeout(10 * time.Second)

			for {
				_, wErr := dl.Write([]byte(
					strconv.FormatUint(uint64(id), 10) + " Hello World UDP"))

				if wErr != nil {
					return
				}

				rLen, rErr := dl.Read(resultData[id].Data[:])

				if rErr != nil {
					t.Error("Failed to write data to testing listener:", rErr)
				}

				resultData[id].Len = rLen

				return
			}
		}(i)
	}

	downWaitGroup.Wait()

	for i := range resultData {
		if string(resultData[i].Data[:resultData[i].Len]) == strconv.FormatUint(
			uint64(i), 10)+" Hello World UDP" {
			continue
		}

		t.Errorf("Failed to communicate with UDP listener. Expecting "+
			"will send and receive data \"%s\", got \"%s\"",
			strconv.FormatUint(uint64(i), 10)+" Hello World UDP",
			string(resultData[i].Data[:resultData[i].Len]))
	}
}
