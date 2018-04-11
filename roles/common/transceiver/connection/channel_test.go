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

package connection

import (
	"bytes"
	"io"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
	"github.com/reinit/coward/common/ticker"
	ch "github.com/reinit/coward/roles/common/channel"
	"github.com/reinit/coward/roles/common/network"
)

type dummyChannelCoder struct{}

type dummyChannelCoderEncoder struct {
	w io.Writer
}

type dummyChannelCoderDecoder struct {
	r io.Reader
}

func (d dummyChannelCoder) Encode(w io.Writer) rw.WriteWriteAll {
	return dummyChannelCoderEncoder{w: w}
}

func (d dummyChannelCoder) Decode(r io.Reader) io.Reader {
	return dummyChannelCoderDecoder{r: r}
}

func (d dummyChannelCoderDecoder) Read(b []byte) (int, error) {
	return d.r.Read(b)
}

func (d dummyChannelCoderEncoder) Write(b []byte) (int, error) {
	return d.WriteAll(b)
}

func (d dummyChannelCoderEncoder) WriteAll(b ...[]byte) (int, error) {
	totalWrite := 0

	for bb := range b {
		wLen, wErr := d.w.Write(b[bb])

		totalWrite += wLen

		if wErr != nil {
			return totalWrite, wErr
		}
	}

	return totalWrite, nil
}

type dummyConnectionWriter struct {
	network.Connection
}

func (d dummyConnectionWriter) SetReadDeadline(t time.Time) error {
	return nil
}

func (d dummyConnectionWriter) Commit() error {
	return nil
}

func (d dummyConnectionWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

type dummyConnection struct {
	network.Connection

	buf *bytes.Buffer
}

func (d *dummyConnection) Closed() <-chan struct{} {
	return nil
}

func (d *dummyConnection) Read(b []byte) (int, error) {
	return d.buf.Read(b)
}

func (d *dummyConnection) Write(b []byte) (int, error) {
	return d.buf.Write(b)
}

func (d *dummyConnection) WriteAll(b ...[]byte) (int, error) {
	totalWrite := 0

	for i := range b {
		wLen, wErr := d.Write(b[i])

		totalWrite += wLen

		if wErr != nil {
			return totalWrite, wErr
		}
	}

	return totalWrite, nil
}

func (d *dummyConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (d *dummyConnection) SetTimeout(t time.Duration) {}

type dummyChannelMachine struct {
	result        func(b []byte)
	conn          Virtual
	segmentReaded int
}

func (d *dummyChannelMachine) Bootup() (fsm.State, error) {
	return d.run, nil
}

func (d *dummyChannelMachine) run(f fsm.FSM) error {
	defer d.conn.Done()

	buf := [1024]byte{}

	for {
		rLen, rErr := d.conn.Read(buf[:])

		if rErr != nil {
			return rErr
		}

		d.result(buf[:rLen])

		// Depleted test AFTER read
		if d.conn.Depleted() {
			d.segmentReaded--

			if d.segmentReaded <= 0 {
				return WrapError(io.EOF)
			}

			return nil
		}
	}
}

func (d *dummyChannelMachine) Shutdown() error {
	return nil
}

func TestChannelReadWrite(t *testing.T) {
	const expectedSegments = 2

	d := &dummyConnection{
		buf: bytes.NewBuffer(make([]byte, 0, 4096)),
	}
	testData := bytes.Repeat(
		[]byte("Hello World"), (math.MaxUint16+3*expectedSegments)/11)

	requestWaitTicker, requestWaitErr := ticker.New(
		300*time.Millisecond, 1024).Serve()

	if requestWaitErr != nil {
		t.Error("Failed to startup Ticker:", requestWaitErr)

		return
	}

	defer requestWaitTicker.Close()

	v := Channelize(d, dummyChannelCoder{}, requestWaitTicker)
	v.Timeout(10 * time.Second)

	defer v.Shutdown()

	wLen, wErr := v.For(8).Write(testData)

	if wErr != nil {
		t.Error("Failed to write due to error:", wErr)

		return
	}

	if wLen != len(testData) {
		t.Errorf("Expecting %d been write in, got %d", len(testData), wLen)

		return
	}

	holdingBuf := make([]byte, 0, 1024)
	reading := true
	chs := ch.New(func(id ch.ID) fsm.Machine {
		return &dummyChannelMachine{
			result: func(b []byte) {
				holdingBuf = append(holdingBuf, b...)
			},
			conn:          v.For(id),
			segmentReaded: expectedSegments,
		}
	}, 255)

	defer chs.Shutdown()

	for reading {
		_, channelMachine, dispatchErr := v.Dispatch(chs)

		if dispatchErr != nil {
			switch dispatchErr.(Error).Get() {
			case io.EOF:

			default:
				t.Error("Dispatch has failed due to error:", dispatchErr)

				return
			}

			break
		}

		var tErr error

		if !channelMachine.Running() {
			tErr = channelMachine.Bootup()
		}

		if tErr == nil {
			tErr = channelMachine.Tick()
		}

		if tErr == nil {
			continue
		}

		switch tErr.(Error).Get() {
		case io.EOF:
			reading = false

		default:
			t.Error("Failed to read due to error:", tErr)

			return
		}
	}

	if !bytes.Equal(holdingBuf, testData) {
		t.Errorf("Failed to Read all written data")

		return
	}
}

type dummyConnection2 struct {
	network.Connection

	bufferedReadData []byte
	readChan         chan []byte
	writeToBuf       *bytes.Buffer
	closed           bool
	closedLock       sync.Mutex
}

func (d *dummyConnection2) Closed() <-chan struct{} {
	return nil
}

func (d *dummyConnection2) Read(b []byte) (int, error) {
	bLen := len(b)
	bufLen := len(d.bufferedReadData)

	if bufLen <= 0 {
		newData, ok := <-d.readChan

		if !ok {
			return 0, io.EOF
		}

		d.bufferedReadData = append(d.bufferedReadData, newData...)
		bufLen = len(d.bufferedReadData)

		if bufLen <= 0 {
			return 0, io.EOF
		}
	}

	cLen := copy(b, d.bufferedReadData)

	if bufLen > bLen {
		d.bufferedReadData = d.bufferedReadData[cLen:]

		return cLen, nil
	}

	d.bufferedReadData = d.bufferedReadData[:0]

	return cLen, nil
}

func (d *dummyConnection2) Write(b []byte) (int, error) {
	return d.writeToBuf.Write(b)
}

func (d *dummyConnection2) WriteAll(b ...[]byte) (int, error) {
	totalWrite := 0

	for i := range b {
		wLen, wErr := d.Write(b[i])

		totalWrite += wLen

		if wErr != nil {
			return totalWrite, wErr
		}
	}

	return totalWrite, nil
}

func (d *dummyConnection2) SetReadDeadline(t time.Time) error {
	return nil
}

func (d *dummyConnection2) SetTimeout(t time.Duration) {}

func (d *dummyConnection2) Close() error {
	d.closedLock.Lock()
	defer d.closedLock.Unlock()

	if d.closed {
		return nil
	}

	close(d.readChan)

	d.closed = true

	return nil
}

type dummyChannelMachine2 struct {
	result func(b []byte)
	conn   Virtual
}

func (d *dummyChannelMachine2) Bootup() (fsm.State, error) {
	return d.run, nil
}

func (d *dummyChannelMachine2) run(f fsm.FSM) error {
	var err error

	wg := sync.WaitGroup{}
	buf := [1024]byte{}

	wg.Add(1)
	go func() {
		defer func() {
			d.conn.Done()
			wg.Done()
		}()

		for {
			rLen, rErr := d.conn.Read(buf[:])

			if rErr != nil {
				err = rErr

				return
			}

			d.result(buf[:rLen])

			if d.conn.Depleted() {
				return
			}
		}
	}()

	wg.Wait()

	return err
}

func (d *dummyChannelMachine2) Shutdown() error {
	return nil
}

func TestChannelDispatchMultipleChannels(t *testing.T) {
	r := make(chan []byte)
	d := &dummyConnection2{
		Connection:       nil,
		bufferedReadData: make([]byte, 0, 1024),
		readChan:         r,
		writeToBuf:       bytes.NewBuffer(make([]byte, 0, 1024)),
		closed:           false,
		closedLock:       sync.Mutex{},
	}

	requestWaitTicker, requestWaitErr := ticker.New(
		300*time.Millisecond, 1024).Serve()

	if requestWaitErr != nil {
		t.Error("Failed to startup Ticker:", requestWaitErr)

		return
	}

	defer requestWaitTicker.Close()

	v := Channelize(d, dummyChannelCoder{}, requestWaitTicker)
	v.Timeout(10 * time.Second)

	defer v.Shutdown()

	resultBuf := [ch.MaxChannels][]byte{}
	vconns := [ch.MaxChannels]Virtual{}

	chs := ch.New(func(id ch.ID) fsm.Machine {
		vconns[id] = v.For(id)

		return &dummyChannelMachine2{
			result: func(b []byte) {
				resultBuf[id] = append(resultBuf[id], b...)
			},
			conn: vconns[id],
		}
	}, 255)

	defer chs.Shutdown()

	waitGroup := sync.WaitGroup{}

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()

		for i := 0; i < 3; i++ {
			for vconnIndex := range vconns {
				_, wErr := vconns[vconnIndex].Write([]byte(
					"HI, THIS IS CHANNEL " +
						strconv.FormatInt(int64(vconnIndex), 10)))

				if wErr == nil {
					continue
				}

				break
			}
		}
	}()

	go func() {
		defer waitGroup.Done()

		for j := 0; j < 10; j++ {
			for i := ch.ID(0); i < ch.MaxChannels; i++ {
				r <- []byte{
					byte(i), 0, 11,
					'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', 'd',
				}
			}
		}
	}()

	go func() {
		defer d.Close()

		waitGroup.Wait()
	}()

	reading := true

	for reading {
		_, channelMachine, dispatchErr := v.Dispatch(chs)

		if dispatchErr != nil {
			break
		}

		var tErr error

		if !channelMachine.Running() {
			tErr = channelMachine.Bootup()
		}

		if tErr == nil {
			tErr = channelMachine.Tick()
		}

		if tErr == nil {
			continue
		}

		break
	}

	for rIdx := range resultBuf {
		if bytes.Equal(
			bytes.Repeat([]byte("Hello World"), 10), resultBuf[rIdx]) {
			continue
		}

		t.Errorf("Failed to read request from Virtual Channel %d. "+
			"Expecting data %d, got %d",
			rIdx, bytes.Repeat([]byte("Hello World"), 10), resultBuf[rIdx])
	}

	writeBufExpected := []byte{}

	for i := 0; i < 3; i++ {
		for vconnIndex := range vconns {
			writeData := "HI, THIS IS CHANNEL " +
				strconv.FormatInt(int64(vconnIndex), 10)
			dataLen := uint16(len(writeData))
			dataLenByte1 := byte(dataLen >> 8)
			dataLenByte2 := byte(dataLen << 8 >> 8)

			wHead := []byte{byte(vconnIndex), dataLenByte1, dataLenByte2}

			writeBufExpected = append(
				writeBufExpected,
				[]byte(wHead)...,
			)

			writeBufExpected = append(
				writeBufExpected,
				[]byte(writeData)...,
			)
		}
	}

	if !bytes.Equal(writeBufExpected, d.writeToBuf.Bytes()) {
		t.Errorf("Failed to write Virtual Channel request to Connection."+
			"Expect writing %d, got %d", writeBufExpected, d.writeToBuf.Bytes())
	}
}

func BenchmarkChannelWrite(b *testing.B) {
	ww := dummyConnectionWriter{}
	vc := channel{
		Connection: ww,
		writeLock:  &sync.Mutex{},
	}
	data := bytes.Repeat([]byte{0}, 4096)

	b.ReportAllocs()
	b.SetBytes(4096)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, wErr := vc.Write(data)

		if wErr == nil {
			continue
		}

		b.Error("Failed to write due to error:", wErr)
	}
}

type dummyBenchmarkConnection struct {
	network.Connection

	preDefinedReadData []byte
}

func (c *dummyBenchmarkConnection) SetReadDeadline(d time.Time) error {
	return nil
}

func (c *dummyBenchmarkConnection) SetDeadline(d time.Time) error {
	return nil
}

func (c *dummyBenchmarkConnection) Closed() <-chan struct{} {
	return nil
}

func (c *dummyBenchmarkConnection) Read(b []byte) (int, error) {
	return copy(b, c.preDefinedReadData), nil
}

func (c *dummyBenchmarkConnection) WriteAll(b ...[]byte) (int, error) {
	return 0, nil
}

func BenchmarkChannelDispatchRead(b *testing.B) {
	testData := bytes.Repeat([]byte{0}, 4096)

	testData[1] = 0x10

	d := &dummyBenchmarkConnection{
		preDefinedReadData: testData,
	}
	v := Channelize(d, dummyChannelCoder{}, nil)
	v.Timeout(10 * time.Second)
	defer v.Shutdown()

	vconns := [ch.MaxChannels]Virtual{}

	chs := ch.New(func(id ch.ID) fsm.Machine {
		vconns[id] = v.For(id)

		return nil
	}, 255)

	rBuf := [4096]byte{}

	b.ReportAllocs()
	b.SetBytes(4096)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, dErr := v.Dispatch(chs)

		if dErr != nil {
			b.Error("Dispatch has failed due to error:", dErr)

			return
		}

		if !func() bool {
			defer vconns[0].Done()

			for {
				_, rErr := vconns[0].Read(rBuf[:])

				if rErr != nil {
					b.Error("Failed to read due to error:", rErr)

					return false
				}

				if vconns[0].Depleted() {
					return true
				}
			}
		}() {
			break
		}
	}
}
