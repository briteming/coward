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

package client

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
)

type dummyConnReading struct {
	Data  []byte
	Error error
}

type dummyDialer struct {
	connReading func() chan dummyConnReading
}

func (d dummyDialer) Dialer() network.Dial {
	return dummyDial{
		connReading: d.connReading,
	}
}

type dummyDial struct {
	connReading func() chan dummyConnReading
}

func (d dummyDial) String() string {
	return "Dummy Dial"
}

func (d dummyDial) Dial() (network.Connection, error) {
	return &dummyConnection{
		Connection: nil,
		reading:    d.connReading(),
	}, nil
}

type dummyConnection struct {
	network.Connection

	reading chan dummyConnReading
}

func (d *dummyConnection) Closed() <-chan struct{} {
	return nil
}

func (d *dummyConnection) SetTimeout(t time.Duration) {}

func (d *dummyConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (d *dummyConnection) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (d *dummyConnection) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (d *dummyConnection) Read(b []byte) (int, error) {
	r, ok := <-d.reading

	if !ok {
		return 0, io.EOF
	}

	return copy(b, r.Data), r.Error
}

func (d *dummyConnection) Close() error {
	select {
	case _, ok := <-d.reading:
		if ok {
			close(d.reading)
		}

		return io.EOF

	default:
		close(d.reading)
	}

	return nil
}

func testDummyEncodec(rw io.ReadWriter) (io.ReadWriter, error) {
	return rw, nil
}

func TestClientUpDown(t *testing.T) {
	log := logger.NewDitch()
	dialer := &dummyDialer{
		connReading: func() chan dummyConnReading {
			return make(chan dummyConnReading, 1)
		},
	}
	requestWaitTicker := time.NewTicker(300 * time.Second)
	defer requestWaitTicker.Stop()
	c := New(0, log, dialer, testDummyEncodec, requestWaitTicker.C, Config{
		MaxConcurrent:        100,
		RequestRetries:       10,
		InitialTimeout:       1 * time.Second,
		IdleTimeout:          3 * time.Second,
		ConnectionPersistent: false,
		ConnectionChannels:   16,
	})

	served, servErr := c.Serve()

	if servErr != nil {
		t.Error("Serve failed due to error:", servErr)

		return
	}

	_, servErr = c.Serve()

	if servErr == nil {
		t.Error("Serve a serving Connector must leads to error")

		return
	}

	closeErr := served.Close()

	if closeErr != nil {
		t.Error("Close failed due to error:", closeErr)

		return
	}

	closeErr = served.Close()

	if closeErr == nil {
		t.Error("Close a closed Connector must leads to error")

		return
	}

	for i := 0; i < 300; i++ {
		served, servErr = c.Serve()

		if servErr != nil {
			t.Error("Serve failed due to error:", servErr)

			return
		}

		closeErr = served.Close()

		if closeErr != nil {
			t.Error("Close failed due to error:", closeErr)

			return
		}
	}
}

type dummyRequestMachine struct {
	Sleep bool
}

func (d dummyRequestMachine) Bootup() (fsm.State, error) {
	return d.run, nil
}

func (d dummyRequestMachine) Ready(f fsm.FSM) error {
	return nil
}

func (d dummyRequestMachine) run(f fsm.FSM) error {
	if d.Sleep {
		time.Sleep(10 * time.Millisecond)
	}

	f.Shutdown()

	return nil
}

func (d dummyRequestMachine) Shutdown() error {
	return nil
}

func dummyRequestBuilder(sleep bool) transceiver.RequestBuilder {
	return func(
		id transceiver.ConnectionID,
		conn rw.ReadWriteDepleteDoner,
		log logger.Logger,
	) fsm.Machine {
		return dummyRequestMachine{
			Sleep: sleep,
		}
	}
}

type dummyMeter struct{}
type dummyMeterStopper struct{}

func (d dummyMeterStopper) Stop() time.Duration {
	return 0
}

func (d dummyMeter) Connection() timer.Stopper {
	return dummyMeterStopper{}
}

func (d dummyMeter) ConnectionFailure(e error) {}

func (d dummyMeter) Request() timer.Stopper {
	return dummyMeterStopper{}
}

func TestClientUpRequestThenDown(t *testing.T) {
	log := logger.NewDitch()
	dialer := &dummyDialer{
		connReading: func() chan dummyConnReading {
			return make(chan dummyConnReading, 1)
		},
	}
	requestWaitTicker := time.NewTicker(300 * time.Second)
	defer requestWaitTicker.Stop()
	c := New(0, log, dialer, testDummyEncodec, requestWaitTicker.C, Config{
		MaxConcurrent:        16,
		RequestRetries:       10,
		InitialTimeout:       1 * time.Second,
		IdleTimeout:          3 * time.Second,
		ConnectionPersistent: false,
		ConnectionChannels:   16,
	})

	serving, servErr := c.Serve()

	if servErr != nil {
		t.Error("Serve failed due to error:", servErr)

		return
	}

	for i := 0; i < 30; i++ {
		go serving.Request(log, dummyRequestBuilder(
			true,
		), nil, dummyMeter{})

		go serving.Request(log, dummyRequestBuilder(
			false,
		), nil, dummyMeter{})
	}

	closeErr := serving.Close()

	if closeErr != nil {
		t.Error("Failed to close due to error:", closeErr)

		return
	}
}

func BenchmarkClientRequest(b *testing.B) {
	log := logger.NewDitch()
	dialer := &dummyDialer{
		connReading: func() chan dummyConnReading {
			return make(chan dummyConnReading, 1)
		},
	}
	requestWaitTicker := time.NewTicker(300 * time.Second)
	defer requestWaitTicker.Stop()
	c := New(0, log, dialer, testDummyEncodec, requestWaitTicker.C, Config{
		MaxConcurrent:        16,
		RequestRetries:       10,
		InitialTimeout:       10 * time.Second,
		IdleTimeout:          10 * time.Second,
		ConnectionPersistent: false,
		ConnectionChannels:   16,
	})

	serving, servErr := c.Serve()

	if servErr != nil {
		b.Error("Serve failed due to error:", servErr)

		return
	}

	defer serving.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, reqErr := serving.Request(log, dummyRequestBuilder(
			false,
		), nil, dummyMeter{})

		if reqErr == nil {
			continue
		}

		b.Error("Request failed due to error: ", reqErr)

		return
	}
}
