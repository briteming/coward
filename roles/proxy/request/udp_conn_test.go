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
	"net"
	"sync"
	"testing"

	"github.com/reinit/coward/roles/common/network/resolve"
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

func (d *dummyUDPConn) ReadFromUDP(rBuf []byte) (int, *net.UDPAddr, error) {
	r, rOK := <-d.Read

	if !rOK {
		return 0, nil, io.EOF
	}

	return copy(rBuf, r.Data), r.Addr, r.Error
}

func (d *dummyUDPConn) WriteToUDP(rBuf []byte, addr *net.UDPAddr) (int, error) {
	dWrite := dummyUDPConnWrite{
		Data: make([]byte, len(rBuf)),
		Addr: addr,
	}

	wLen := copy(dWrite.Data, rBuf)

	d.Write <- dWrite

	return wLen, nil
}

func (d *dummyUDPConn) Close() error {
	return nil
}

type dummyResolver struct {
	resolves map[string][]net.IP
}

var (
	errdummyResolverFailed = errors.New("Resolve Failed")
)

func (d *dummyResolver) Resolve(host string) ([]net.IP, error) {
	ips, found := d.resolves[host]

	if !found {
		return nil, errdummyResolverFailed
	}

	return ips, nil
}

func (d *dummyResolver) Reverse(ipi net.IP) (string, error) {
	for k, v := range d.resolves {
		for _, ip := range v {
			if !ip.Equal(ipi) {
				continue
			}

			return k, nil
		}
	}

	return "", errdummyResolverFailed
}

func testUDPConnRead(
	t *testing.T,
	read chan dummyUDPConnRead,
	resolves map[string][]net.IP,
	recorded map[resolve.IPMark]struct{},
) ([]byte, error) {
	uc := udpConn{
		UDPConn:    &dummyUDPConn{Read: read, Write: nil},
		resolver:   &dummyResolver{resolves: resolves},
		remotes:    recorded,
		maxRemotes: 16,
		remoteLock: sync.RWMutex{},
	}

	buf := [4096]byte{}

	rLen, rErr := uc.Read(buf[:])

	if rErr != nil {
		return nil, rErr
	}

	return buf[:rLen], nil
}

func TestUDPConnReadIPv4(t *testing.T) {
	rChan := make(chan dummyUDPConnRead, 1)

	rChan <- dummyUDPConnRead{
		Data: []byte("AAABBB!!!"),
		Addr: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.2"),
			Port: 198,
			Zone: "",
		},
		Error: nil,
	}

	ipMark := resolve.IPMark{}
	ipMark.Import(net.ParseIP("127.0.0.2"))

	readed, readErr := testUDPConnRead(
		t,
		rChan,
		map[string][]net.IP{},
		map[resolve.IPMark]struct{}{
			ipMark: struct{}{},
		})

	if readErr != nil {
		t.Error("Failed to read due to error:", readErr)

		return
	}

	if string(readed) != string([]byte{UDPSendIPv4, 127, 0, 0, 2, 0, 198, 65,
		65, 65, 66, 66, 66, 33, 33, 33}) {
		t.Errorf("Failed to read expected data. Expecting data to "+
			"be %d, got %d", readed, []byte{UDPSendIPv4, 127, 0, 0, 2, 0, 198,
			65, 65, 65, 66, 66, 66, 33, 33, 33})

		return
	}
}

func TestUDPConnReadIPv6(t *testing.T) {
	rChan := make(chan dummyUDPConnRead, 1)

	rChan <- dummyUDPConnRead{
		Data: []byte("AAABBB!!!"),
		Addr: &net.UDPAddr{
			IP:   net.ParseIP("::1"),
			Port: 198,
			Zone: "",
		},
		Error: nil,
	}

	ipMark := resolve.IPMark{}
	ipMark.Import(net.ParseIP("::1"))

	readed, readErr := testUDPConnRead(
		t,
		rChan,
		map[string][]net.IP{},
		map[resolve.IPMark]struct{}{
			ipMark: struct{}{},
		})

	if readErr != nil {
		t.Error("Failed to read due to error:", readErr)

		return
	}

	if string(readed) != string([]byte{UDPSendIPv6, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 1, 0, 198, 65, 65, 65, 66, 66, 66, 33, 33, 33}) {
		t.Errorf("Failed to read expected data. Expecting data to "+
			"be %d, got %d", readed, []byte{UDPSendIPv6, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 1, 0, 198, 65, 65, 65, 66, 66, 66, 33, 33, 33})

		return
	}
}

func TestUDPConnReadHost(t *testing.T) {
	rChan := make(chan dummyUDPConnRead, 1)

	rChan <- dummyUDPConnRead{
		Data: []byte("AAABBB!!!"),
		Addr: &net.UDPAddr{
			IP:   net.ParseIP("::1"),
			Port: 198,
			Zone: "",
		},
		Error: nil,
	}

	ipMark := resolve.IPMark{}
	ipMark.Import(net.ParseIP("::1"))

	readed, readErr := testUDPConnRead(
		t,
		rChan,
		map[string][]net.IP{
			"localhost": []net.IP{net.ParseIP("::1")},
		},
		map[resolve.IPMark]struct{}{
			ipMark: struct{}{},
		})

	if readErr != nil {
		t.Error("Failed to read due to error:", readErr)

		return
	}

	if string(readed) != string([]byte{UDPSendHost, 9, 108, 111, 99, 97, 108,
		104, 111, 115, 116, 0, 198, 65, 65, 65, 66, 66, 66, 33, 33, 33}) {
		t.Errorf("Failed to read expected data. Expecting data to "+
			"be %d, got %d", readed, []byte{UDPSendHost, 9, 108, 111, 99, 97,
			108, 104, 111, 115, 116, 0, 198, 65, 65, 65, 66, 66, 66, 33, 33,
			33})

		return
	}
}

func testUDPConnWrite(
	t *testing.T,
	res map[string][]net.IP,
	data []byte,
) (
	int,
	dummyUDPConnWrite,
	map[string][]net.IP,
	map[resolve.IPMark]struct{},
	error,
) {
	rec := map[resolve.IPMark]struct{}{}
	wcc := make(chan dummyUDPConnWrite, 1)
	ucn := udpConn{
		UDPConn:    &dummyUDPConn{Read: nil, Write: wcc},
		resolver:   &dummyResolver{resolves: res},
		remotes:    rec,
		maxRemotes: 16,
		remoteLock: sync.RWMutex{},
	}

	wLen, wErr := ucn.Write(data)

	if wErr != nil {
		return wLen, dummyUDPConnWrite{}, nil, nil, wErr
	}

	return wLen, <-wcc, res, rec, wErr
}

func TestUDPConnWriteIPv4(t *testing.T) {
	_, rData, _, recordedHost, rErr := testUDPConnWrite(
		t,
		map[string][]net.IP{},
		[]byte{UDPSendIPv4, 128, 0, 0, 33, 1, 80, 'H', 'E', 'L', 'L', 'O'})

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if string(rData.Data) != "HELLO" {
		t.Error("Failed to read expected data \"HELLO\", got:", rData)

		return
	}

	expectedIP := net.ParseIP("128.0.0.33")
	ipMark := resolve.IPMark{}
	ipMark.Import(expectedIP)

	_, recorded := recordedHost[ipMark]

	if !recorded {
		t.Error("Expecting the host 128.0.0.33 will be added to the record, " +
			"and it's not what's happening")

		return
	}

	if !rData.Addr.IP.Equal(expectedIP) {
		t.Error("Expecting the data was coming from 127.0.0.33, got:",
			rData.Addr.IP)

		return
	}
}

func TestUDPConnWriteIPv6(t *testing.T) {
	_, rData, _, recordedHost, rErr := testUDPConnWrite(
		t,
		map[string][]net.IP{},
		[]byte{UDPSendIPv6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 51,
			1, 80, 'H', 'E', 'L', 'L', 'O'})

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if string(rData.Data) != "HELLO" {
		t.Error("Failed to read expected data \"HELLO\", got:", rData)

		return
	}

	expectedIP := net.ParseIP("::33")
	ipMark := resolve.IPMark{}
	ipMark.Import(expectedIP)

	_, recorded := recordedHost[ipMark]

	if !recorded {
		t.Error("Expecting the host ::33 will be added to the record, " +
			"and it's not what's happening")

		return
	}

	if !rData.Addr.IP.Equal(expectedIP) {
		t.Error("Expecting the data was coming from ::33, got:",
			rData.Addr.IP)

		return
	}
}

func TestUDPConnWriteHost(t *testing.T) {
	_, rData, dnsResolved, recordedHost, rErr := testUDPConnWrite(
		t,
		map[string][]net.IP{
			"localhost": []net.IP{net.ParseIP("128.0.0.1")},
		},
		[]byte{
			UDPSendHost, 9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't',
			1, 80, 'H', 'E', 'L', 'L', 'O'})

	if rErr != nil {
		t.Error("Failed to read due to error:", rErr)

		return
	}

	if string(rData.Data) != "HELLO" {
		t.Error("Failed to read expected data \"HELLO\", got:", rData)

		return
	}

	expectedIP := net.ParseIP("128.0.0.1")
	ipMark := resolve.IPMark{}
	ipMark.Import(expectedIP)

	_, recorded := recordedHost[ipMark]

	if !recorded {
		t.Error("Expecting the host ::33 will be added to the record, " +
			"and it's not what's happening")

		return
	}

	if !rData.Addr.IP.Equal(expectedIP) {
		t.Error("Expecting the data was coming from ::33, got:",
			rData.Addr.IP)

		return
	}

	_, beenResolved := dnsResolved["localhost"]

	if !beenResolved {
		t.Error("Expecting \"localhost\" will be resolved, " +
			"that's is not what happened over there")

		return
	}
}
