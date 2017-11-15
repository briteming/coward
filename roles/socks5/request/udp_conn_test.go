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
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/roles/proxy/request"
	"github.com/reinit/coward/roles/socks5/common"
)

type dummyUDPConnRead struct {
	Data  []byte
	Addr  *net.UDPAddr
	Error error
}
type dummyUDPConnWrite struct {
	Data []byte
	Addr *net.UDPAddr
}

type dummyUDPConn struct {
	Read  chan dummyUDPConnRead
	Write chan dummyUDPConnWrite
}

func (d dummyUDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	r, rOK := <-d.Read

	if !rOK {
		return 0, nil, io.EOF
	}

	return copy(b, r.Data), r.Addr, r.Error
}

func (d dummyUDPConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	dataWrite := dummyUDPConnWrite{
		Data: b,
		Addr: addr,
	}

	d.Write <- dataWrite

	return len(b), nil
}

func (d dummyUDPConn) Close() error {
	return nil
}

func testUDPConnRead(
	t *testing.T,
	read chan dummyUDPConnRead,
	allowedIP net.IP,
) ([]byte, error) {
	udpC := udpConn{
		UDPConn: dummyUDPConn{
			Read:  read,
			Write: nil,
		},
		allowedIP: allowedIP,
		client:    nil,
	}

	buf := [4096]byte{}

	rLen, rErr := udpC.Read(buf[:])

	if rErr != nil {
		return nil, rErr
	}

	return buf[:rLen], nil
}

func TestUDPConnReadIPv4(t *testing.T) {
	rChan := make(chan dummyUDPConnRead, 1)

	rChan <- dummyUDPConnRead{
		Data: []byte{0, 0, 0, byte(common.ATypeIPv4), 127, 0, 0, 1, 0, 198,
			'H', 'E', 'L', 'L', 'O', '!'},
		Addr: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.2"),
			Port: 198,
			Zone: "",
		},
		Error: nil,
	}

	readData, readErr := testUDPConnRead(t, rChan, net.ParseIP("127.0.0.2"))

	if readErr != nil {
		t.Error("Failed to read data due to error:", readErr)

		return
	}

	if !bytes.Equal(readData, []byte{byte(request.UDPSendIPv4), 127, 0, 0, 1,
		0, 198, 72, 69, 76, 76, 79, 33}) {
		t.Errorf("Failed to convert data in to specified format. "+
			"Expecting %d, got %d", readData, []byte{byte(request.UDPSendIPv4),
			127, 0, 0, 1, 0, 198, 72, 69, 76, 76, 79, 33})

		return
	}
}

func TestUDPConnReadIPv6(t *testing.T) {
	rChan := make(chan dummyUDPConnRead, 1)

	rChan <- dummyUDPConnRead{
		Data: []byte{0, 0, 0, byte(common.ATypeIPv6),
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 198,
			'H', 'E', 'L', 'L', 'O', '!'},
		Addr: &net.UDPAddr{
			IP:   net.ParseIP("::2"),
			Port: 198,
			Zone: "",
		},
		Error: nil,
	}

	readData, readErr := testUDPConnRead(t, rChan, net.ParseIP("::2"))

	if readErr != nil {
		t.Error("Failed to read data due to error:", readErr)

		return
	}

	if !bytes.Equal(readData, []byte{byte(request.UDPSendIPv6),
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 198,
		72, 69, 76, 76, 79, 33}) {
		t.Errorf("Failed to convert data in to specified format. "+
			"Expecting %d, got %d", readData, []byte{byte(request.UDPSendIPv6),
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 198,
			72, 69, 76, 76, 79, 33})

		return
	}
}

func TestUDPConnReadHost(t *testing.T) {
	rChan := make(chan dummyUDPConnRead, 1)

	rChan <- dummyUDPConnRead{
		Data: []byte{0, 0, 0, byte(common.ATypeHost),
			9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0, 198,
			'H', 'E', 'L', 'L', 'O', '!'},
		Addr: &net.UDPAddr{
			IP:   net.ParseIP("::2"),
			Port: 198,
			Zone: "",
		},
		Error: nil,
	}

	readData, readErr := testUDPConnRead(t, rChan, net.ParseIP("::2"))

	if readErr != nil {
		t.Error("Failed to read data due to error:", readErr)

		return
	}

	if !bytes.Equal(readData, []byte{byte(request.UDPSendHost),
		9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0, 198,
		72, 69, 76, 76, 79, 33}) {
		t.Errorf("Failed to convert data in to specified format. "+
			"Expecting %d, got %d", readData, []byte{byte(request.UDPSendHost),
			9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0, 198,
			72, 69, 76, 76, 79, 33})

		return
	}
}

func testUDPConnWrite(
	t *testing.T,
	data []byte,
) (int, []byte, *net.UDPAddr, error) {
	write := make(chan dummyUDPConnWrite, 1)

	udpC := &udpConn{
		UDPConn: dummyUDPConn{
			Read:  nil,
			Write: write,
		},
		allowedIP: nil,
		client: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.22"),
			Port: 8888,
			Zone: "",
		},
	}

	wLen, wErr := rw.WriteFull(udpC, data)

	wData := <-write

	return wLen, wData.Data, wData.Addr, wErr
}

func TestUDPConnWriteIPv4(t *testing.T) {
	_, result, _, wErr := testUDPConnWrite(t, []byte{request.UDPSendIPv4,
		127, 0, 0, 1, 0, 198, 'H', 'E', 'L', 'L', 'O', '!'})

	if wErr != nil {
		t.Error("Failed to write to UDPConn due to error:", wErr)

		return
	}

	if !bytes.Equal(result, []byte{0, 0, 0, byte(common.ATypeIPv4),
		127, 0, 0, 1, 0, 198, 'H', 'E', 'L', 'L', 'O', '!'}) {
		t.Errorf("Failed to write to UDPConn. Expecting %d will be written"+
			", got %d", result, []byte{0, 0, 0, byte(common.ATypeIPv4),
			127, 0, 0, 1, 0, 198, 'H', 'E', 'L', 'L', 'O', '!'})

		return
	}
}

func TestUDPConnWriteIPv6(t *testing.T) {
	_, result, _, wErr := testUDPConnWrite(t, []byte{request.UDPSendIPv6,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 198,
		'H', 'E', 'L', 'L', 'O', '!'})

	if wErr != nil {
		t.Error("Failed to write to UDPConn due to error:", wErr)

		return
	}

	if !bytes.Equal(result, []byte{0, 0, 0, byte(common.ATypeIPv6),
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 198,
		'H', 'E', 'L', 'L', 'O', '!'}) {
		t.Errorf("Failed to write to UDPConn. Expecting %d will be written"+
			", got %d", result, []byte{0, 0, 0, byte(common.ATypeIPv6),
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 198,
			'H', 'E', 'L', 'L', 'O', '!'})

		return
	}
}

func TestUDPConnWriteHost(t *testing.T) {
	_, result, _, wErr := testUDPConnWrite(t, []byte{request.UDPSendHost,
		9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0, 198,
		'H', 'E', 'L', 'L', 'O', '!'})

	if wErr != nil {
		t.Error("Failed to write to UDPConn due to error:", wErr)

		return
	}

	if !bytes.Equal(result, []byte{0, 0, 0, byte(common.ATypeHost),
		9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0, 198,
		'H', 'E', 'L', 'L', 'O', '!'}) {
		t.Errorf("Failed to write to UDPConn. Expecting %d will be written"+
			", got %d", result, []byte{0, 0, 0, byte(common.ATypeHost),
			9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0, 198,
			'H', 'E', 'L', 'L', 'O', '!'})

		return
	}
}
