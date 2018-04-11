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

package join

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/ticker"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/relay"
	"github.com/reinit/coward/roles/projector/projection"
)

type dummyProjections struct {
	projections map[projection.ID]*dummyProjection
}

func (d *dummyProjections) Projection(
	id projection.ID) (projection.Projection, error) {
	pp, ppFound := d.projections[id]

	if !ppFound {
		return nil, errors.New("Projection not found")
	}

	return pp, nil
}

func (d *dummyProjections) Handler(id projection.ID) (network.Handler, error) {
	return nil, nil
}

type dummyAccessorResult struct {
	err       error
	retriable bool
	reset     bool
	wait      chan struct{}
}

type dummyAccessor struct {
	access network.Connection
	result chan dummyAccessorResult
	rr     worker.Runner
}

func (d dummyAccessor) Access() network.Connection {
	return d.access
}

func (d dummyAccessor) Result(
	e error, retriable bool, reset bool, wait chan struct{}) {
	d.result <- dummyAccessorResult{
		err: e, retriable: retriable, reset: reset, wait: wait}
}

func (d dummyAccessor) Runner() worker.Runner {
	return d.rr
}

func (d dummyAccessor) Proccessor(p projection.Proccessor) {}

type dummyProjection struct {
	id         projection.ID
	accessChan chan projection.Accessor
}

func (d *dummyProjection) Receive(c network.Connection) error {
	aa := dummyAccessor{
		access: c,
		result: make(chan dummyAccessorResult),
	}

	d.accessChan <- aa

	result := <-aa.result

	return result.err
}

func (d *dummyProjection) Receiver() projection.Receiver {
	return &dummyProjectionReceiver{
		id:         d.id,
		accessChan: d.accessChan,
	}
}

type dummyProjectionReceiver struct {
	id         projection.ID
	accessChan chan projection.Accessor
}

func (d *dummyProjectionReceiver) ID() projection.ID {
	return d.id
}

func (d *dummyProjectionReceiver) Receive() chan projection.Accessor {
	return d.accessChan
}

func (d *dummyProjectionReceiver) Delay(delay time.Duration) {}
func (d *dummyProjectionReceiver) Expand()                   {}
func (d *dummyProjectionReceiver) Shrink()                   {}
func (d *dummyProjectionReceiver) Remove()                   {}
func (d *dummyProjectionReceiver) ShrinkAndRemove()          {}

func testGetJoin() (*join, *dummyProjection, *dummyProjection, worker.Runner) {
	tk, tkErr := ticker.New(300, 1024).Serve()

	if tkErr != nil {
		panic(fmt.Sprintf("Failed to create ticker due to error: %s", tkErr))
	}

	rr, rrErr := worker.New(logger.NewDitch(), tk, worker.Config{
		MaxWorkers:        2,
		MinWorkers:        2,
		MaxWorkerIdle:     16 * time.Second,
		JobReceiveTimeout: 10 * time.Second,
	}).Serve()

	if rrErr != nil {
		panic(rrErr)
	}

	cfg := Config{
		ConnectionID:    "",
		ConnectionDelay: timer.New(),
		Buffer:          make([]byte, 4096),
		Timeout:         1,
	}

	reqReceiveTick := time.NewTicker(10 * time.Millisecond)
	defer reqReceiveTick.Stop()

	dp1 := &dummyProjection{
		id:         10,
		accessChan: make(chan projection.Accessor),
	}
	dp2 := &dummyProjection{
		id:         12,
		accessChan: make(chan projection.Accessor),
	}

	proj := &dummyProjections{
		projections: map[projection.ID]*dummyProjection{
			10: dp1,
			12: dp2,
		},
	}

	return &join{
		cfg:    cfg,
		runner: rr,
		registered: registerations{
			projections: proj,
			receivers: make(
				map[projection.ID]registeration, projection.MaxID),
		},
	}, dp1, dp2, rr
}

type dummyReadWriteDoner struct {
	readChan      chan *bytes.Buffer
	currentReader *bytes.Buffer
	written       *bytes.Buffer
	writeLock     sync.Mutex
}

func (d *dummyReadWriteDoner) Read(b []byte) (int, error) {
	if d.currentReader == nil || d.currentReader.Len() <= 0 {
		currentReader, ok := <-d.readChan

		if !ok {
			return 0, io.EOF
		}

		d.currentReader = currentReader
	}

	return d.currentReader.Read(b)
}

func (d *dummyReadWriteDoner) Write(b []byte) (int, error) {
	d.writeLock.Lock()
	defer d.writeLock.Unlock()

	return d.written.Write(b)
}

func (d *dummyReadWriteDoner) WaitWrite() {
	for {
		time.Sleep(5 * time.Millisecond)

		d.writeLock.Lock()

		if d.written.Len() <= 0 {
			d.writeLock.Unlock()

			continue
		}

		d.writeLock.Unlock()

		break
	}
}

func (d *dummyReadWriteDoner) Depleted() bool {
	return d.currentReader == nil || d.currentReader.Len() <= 0
}

func (d *dummyReadWriteDoner) Deplete() error {
	if d.currentReader != nil {
		d.currentReader.Reset()
		d.currentReader = nil
	}

	return nil
}

func (d *dummyReadWriteDoner) Done() error {
	if d.Depleted() {
		return nil
	}

	return d.Deplete()
}

type dummyNetworkConnection struct {
	network.Connection

	rw *dummyReadWriteDoner
}

func (d *dummyNetworkConnection) Read(b []byte) (int, error) {
	return d.rw.Read(b)
}

func (d *dummyNetworkConnection) Write(b []byte) (int, error) {
	return d.rw.Write(b)
}

func (d *dummyNetworkConnection) SetTimeout(t time.Duration) {}

func (d *dummyNetworkConnection) Close() error {
	return nil
}

func TestProccessor(t *testing.T) {
	j, dp1, dp2, rr := testGetJoin()
	clientConn := &dummyReadWriteDoner{
		readChan:      make(chan *bytes.Buffer),
		currentReader: nil,
		written:       bytes.NewBuffer(make([]byte, 0, 4096)),
	}

	_ = dp2

	waitG := sync.WaitGroup{}

	waitG.Add(1)

	go func() {
		defer func() {
			close(clientConn.readChan)
			waitG.Done()
		}()

		for i := 0; i < 10; i++ {
			// Client: Register as a receiver for projection 10
			clientConn.readChan <- bytes.NewBuffer([]byte{10})

			// Server: OK, you have register projection 10, must heartbeat
			// with in 1 second
			clientConn.WaitWrite()

			if !bytes.Equal(
				clientConn.written.Bytes(), []byte{RespondJoined, 0, 1}) {
				t.Errorf("Expecting RespondJoined %d, got %d",
					[]byte{RespondJoined, 0, 1}, clientConn.written.Bytes())

				return
			}

			clientConn.written.Reset()

			for j := 0; j < 10; j++ {
				// Server: Someone is accessing projection 10
				accConn := &dummyNetworkConnection{
					Connection: nil,
					rw: &dummyReadWriteDoner{
						readChan:      make(chan *bytes.Buffer),
						currentReader: nil,
						written:       bytes.NewBuffer(make([]byte, 0, 4096)),
					},
				}
				acc := dummyAccessor{
					access: accConn,
					result: make(chan dummyAccessorResult),
					rr:     rr,
				}

				dp1.accessChan <- acc

				clientConn.WaitWrite()

				if !bytes.Equal(
					clientConn.written.Bytes(),
					[]byte{RequestClientRelayRequest}) {
					t.Errorf("Expecting RequestClientRelayInit %d, got %d",
						[]byte{RequestClientRelayRequest},
						clientConn.written.Bytes())

					return
				}

				clientConn.written.Reset()

				// Client: Relay is ready
				clientConn.readChan <- bytes.NewBuffer([]byte{
					RespondClientRelayInitialized, 0})

				// Server relaying data to the client relay
				accConn.rw.readChan <- bytes.NewBuffer([]byte("Hello World"))

				clientConn.WaitWrite()

				if !bytes.Equal(clientConn.written.Bytes(),
					[]byte{byte(relay.SignalData),
						'H', 'e', 'l', 'l', 'o', ' ',
						'W', 'o', 'r', 'l', 'd'}) {
					t.Errorf("Expecting relay.SignalData %d, got %d",
						[]byte{byte(relay.SignalData),
							'H', 'e', 'l', 'l', 'o', ' ',
							'W', 'o', 'r', 'l', 'd'},
						clientConn.written.Bytes())

					return
				}

				clientConn.written.Reset()

				// Client relaying data to server relay
				clientConn.readChan <- bytes.NewBuffer([]byte{
					byte(relay.SignalData),
					'H', 'e', 'l', 'l', 'o', ' ',
					'W', 'o', 'r', 'l', 'd'})

				accConn.rw.WaitWrite()

				if !bytes.Equal(accConn.rw.written.Bytes(),
					[]byte{'H', 'e', 'l', 'l', 'o', ' ',
						'W', 'o', 'r', 'l', 'd'}) {
					t.Errorf("Expecting relay.SignalData %d, got %d",
						[]byte{'H', 'e', 'l', 'l', 'o', ' ',
							'W', 'o', 'r', 'l', 'd'},
						accConn.rw.written.Bytes())

					return
				}

				accConn.rw.written.Reset()

				// Server: accessor is disconnected
				close(accConn.rw.readChan)

				clientConn.WaitWrite()

				if !bytes.Equal(clientConn.written.Bytes(), []byte{byte(
					relay.SignalCompleted)}) {
					t.Errorf("Expecting relay.SignalCompleted %d, got %d",
						[]byte{byte(relay.SignalCompleted)},
						clientConn.written.Bytes())

					return
				}

				clientConn.written.Reset()

				// Client: OK, I have closed my side of relay
				clientConn.readChan <- bytes.NewBuffer([]byte{
					byte(relay.SignalClosed)})

				accResult := <-acc.result

				if accResult.wait != nil {
					<-accResult.wait
				}

				if accResult.err != nil {
					t.Errorf("Expecting no error after relay close, got %s",
						accResult.err)

					return
				}

				if accResult.retriable {
					t.Errorf("There must be no Retry after relay close")

					return
				}

				if accResult.reset {
					t.Errorf("There must be no Reset after relay close")

					return
				}
			}

			clientConn.readChan <- bytes.NewBuffer([]byte{
				RespondClientRelease})

			clientConn.WaitWrite()

			if !bytes.Equal(clientConn.written.Bytes(),
				[]byte{RequestClientRelease}) {
				t.Errorf("Expecting RequestClientQuitConfirm %d, got %d",
					[]byte{RequestClientRelease},
					clientConn.written.Bytes())

				return
			}

			clientConn.written.Reset()

			clientConn.readChan <- bytes.NewBuffer([]byte{
				RespondClientQuit})
		}
	}()

	machine := fsm.New(j.New(clientConn, logger.NewDitch()))
	bootErr := machine.Bootup()

	if bootErr != nil {
		t.Error("Failed to bootup:", bootErr)

		waitG.Wait()

		return
	}

	waitG.Add(1)

	go func() {
		defer waitG.Done()

		testTimes := 10

		for {
			tkErr := machine.Tick()

			if tkErr != nil {
				t.Error("Proccessor encountered an error during proccessing:",
					tkErr)

				return
			}

			if machine.Running() {
				continue
			}

			testTimes--

			if testTimes <= 0 {
				return
			}

			bootErr := machine.Bootup()

			if bootErr != nil {
				t.Error("Failed to bootup:", bootErr)

				return
			}
		}
	}()

	waitG.Wait()
}
