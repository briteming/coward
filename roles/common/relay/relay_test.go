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

package relay

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/reinit/coward/common/corunner"
	"github.com/reinit/coward/common/logger"
)

type dummyServerConn struct {
	r         chan io.Reader
	w         io.Writer
	eofed     bool
	currentRR io.Reader
}

func (d *dummyServerConn) Read(b []byte) (int, error) {
	if d.eofed {
		return 0, io.EOF
	}

	var rr io.Reader

	if d.currentRR != nil {
		rr = d.currentRR
	} else {
		rr = <-d.r

		d.currentRR = rr
	}

	rLen, rErr := rr.Read(b)

	if rErr == io.EOF {
		d.currentRR = nil
		d.eofed = true

		return rLen, nil
	}

	return rLen, rErr
}

func (d *dummyServerConn) Write(b []byte) (int, error) {
	return d.w.Write(b)
}

func (d *dummyServerConn) Deplete() error {
	return nil
}

func (d *dummyServerConn) Depleted() bool {
	return d.eofed
}

func (d *dummyServerConn) Done() error {
	d.eofed = false

	return nil
}

type dummyClient1 struct {
	r chan io.Reader
	w io.ReadWriter
}

func (d *dummyClient1) Read(b []byte) (int, error) {
	rr, ok := <-d.r

	if !ok {
		return 0, io.EOF
	}

	if rr == nil {
		return 0, io.EOF
	}

	rLen, rErr := rr.Read(b)

	if rErr != nil {
		return 0, nil
	}

	return rLen, rErr
}

func (d *dummyClient1) Write(b []byte) (int, error) {
	return d.w.Write(b)
}

func (d *dummyClient1) Close() error {
	select {
	case <-d.r:

	default:
		close(d.r)
	}

	return nil
}

func (d *dummyClient1) SendSignal(s Signal, e Signal) error {
	return nil
}

type dummyClientBuilder1 struct {
	conn io.ReadWriteCloser
}

func (d *dummyClientBuilder1) Initialize(
	server Server,
) error {
	return nil
}

func (d *dummyClientBuilder1) Client(
	server Server,
) (io.ReadWriteCloser, error) {
	return d.conn, nil
}

type dummyCountedBufferWrite struct {
	w     *bytes.Buffer
	count chan struct{}
}

func (d *dummyCountedBufferWrite) Write(b []byte) (int, error) {
	wLen, wErr := d.w.Write(b)

	<-d.count

	return wLen, wErr
}

func TestRelayRelay(t *testing.T) {
	runner, runnerErr := corunner.New(logger.NewDitch(), corunner.Config{
		MaxWorkers:        64,
		MinWorkers:        64,
		MaxWorkerIdle:     5 * time.Minute,
		JobReceiveTimeout: 300 * time.Millisecond,
	}).Serve()

	if runnerErr != nil {
		t.Error("Failed to start Relay runner:", runnerErr)

		return
	}

	clientBuffer1 := [4096]byte{}
	serverBuffer1 := [4096]byte{}
	relay1Reading := make(chan io.Reader, 1)
	relay1Sends := &dummyCountedBufferWrite{
		w:     bytes.NewBuffer(make([]byte, 0, 4096)),
		count: make(chan struct{}, 1),
	}
	relay1ClientReading := make(chan io.Reader)
	relay1ClientSends := bytes.NewBuffer(make([]byte, 0, 4096))

	relay1 := New(runner, &dummyServerConn{
		r: relay1Reading,
		w: relay1Sends,
	}, clientBuffer1[:], &dummyClientBuilder1{
		conn: &dummyClient1{
			r: relay1ClientReading,
			w: relay1ClientSends,
		},
	}, serverBuffer1[:])

	bootupErr := relay1.Bootup(nil)

	if bootupErr != nil {
		t.Error("Failed to boot up Relay 1 due to error:", bootupErr)

		return
	}

	relay1Sends.count <- struct{}{}
	relay1ClientReading <- bytes.NewBuffer([]byte("Test "))

	relay1Sends.count <- struct{}{}
	relay1ClientReading <- bytes.NewBuffer([]byte("data"))

	relay1Sends.count <- struct{}{}
	relay1ClientReading <- nil

	relay1Sends.count <- struct{}{}

	if !bytes.Equal(
		[]byte{
			byte(SignalData),
			84, 101, 115, 116, 32,
			byte(SignalData),
			100, 97, 116, 97,
			byte(SignalCompleted)},
		relay1Sends.w.Bytes(),
	) {
		t.Errorf("Expect the Relay will send %d, got %d",
			[]byte{
				byte(SignalData),
				84, 101, 115, 116, 32,
				byte(SignalData),
				100, 97, 116, 97,
				byte(SignalCompleted)},
			relay1Sends.w.Bytes())

		return
	}

	if !relay1.(*relay).running {
		t.Error("Relay 1 has been shutted down unexpectly")

		return
	}

	clientBuffer2 := [4096]byte{}
	serverBuffer2 := [4096]byte{}
	relay2Reading := make(chan io.Reader, 1)
	relay2Sends := &dummyCountedBufferWrite{
		w:     bytes.NewBuffer(make([]byte, 0, 4096)),
		count: make(chan struct{}, 1),
	}
	relay2ClientReading := make(chan io.Reader)
	relay2ClientSends := bytes.NewBuffer(make([]byte, 0, 4096))

	relay2 := New(runner, &dummyServerConn{
		r: relay2Reading,
		w: relay2Sends,
	}, clientBuffer2[:], &dummyClientBuilder1{
		conn: &dummyClient1{
			r: relay2ClientReading,
			w: relay2ClientSends,
		},
	}, serverBuffer2[:])

	bootupErr = relay2.Bootup(nil)

	if bootupErr != nil {
		t.Error("Failed to boot up Relay 2 due to error:", bootupErr)

		return
	}

	relay2Reading <- bytes.NewBuffer([]byte{
		byte(SignalData), 84, 101, 115, 116, 32})
	tickErr := relay2.Tick()

	if tickErr != nil {
		t.Error("Relay 2 has failed to tick due to error:", tickErr)

		return
	}

	relay2Reading <- bytes.NewBuffer([]byte{
		byte(SignalData), 100, 97, 116, 97})
	tickErr = relay2.Tick()

	if tickErr != nil {
		t.Error("Relay 2 has failed to tick due to error:", tickErr)

		return
	}

	relay2Reading <- bytes.NewBuffer([]byte{byte(SignalClose)})
	relay2Sends.count <- struct{}{}
	tickErr = relay2.Tick()

	if tickErr != nil {
		t.Error("Relay 2 has failed to tick due to error:", tickErr)

		return
	}

	if !bytes.Equal(
		[]byte{84, 101, 115, 116, 32, 100, 97, 116, 97},
		relay2ClientSends.Bytes(),
	) {
		t.Errorf("Expect the Relay will deliver %d, got %d",
			[]byte{84, 101, 115, 116, 32, 100, 97, 116, 97},
			relay2ClientSends.Bytes())

		return
	}

	closeErr := relay2.Close()

	if closeErr == nil {
		t.Error("Expecting an error while closing Relay 2, " +
			"as it should already be closed")

		return
	}

	relay1Reading <- bytes.NewBuffer([]byte{byte(SignalClose)})
	tickErr = relay1.Tick()

	if tickErr != nil {
		t.Error("Relay 1 has failed to tick due to error:", tickErr)

		return
	}

	if relay1.(*relay).running {
		t.Error("Relay 1 must be shutted down now")

		return
	}
}
