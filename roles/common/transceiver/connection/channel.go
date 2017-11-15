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

package connection

import (
	"io"
	"math"
	"sync"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/rw"
	ch "github.com/reinit/coward/roles/common/channel"
	"github.com/reinit/coward/roles/common/network"
)

// Errors
var (
	ErrChannelShuttedDown = NewError(
		"Connection Channel already closed")

	ErrChannelInitializationFailed = NewError(
		"Failed to read initial information of Connection Channel")

	ErrChannelWriteFailedToCopyData = NewError(
		"Failed to copy data for writing")

	ErrChannelWriteIncomplete = NewError(
		"Incomplete write")

	ErrChannelDispatchChannelUnavailable = NewError(
		"Selected Channel is unavailable")

	ErrChannelDispatchChannelInactive = NewError(
		"Selected Channel is inactive")

	ErrChannelVirtualConnectionTimedout = NewError(
		"Virtual Connection has timed out")

	ErrChannelVirtualConnectionSegmentDepleted = NewError(
		"Data Segment of current Virtual Connection are depleted")

	ErrChannelConnectionDropped = NewError(
		"Channel Connection is lost")

	ErrChannelVirtualConnectionNotAssigned = NewError(
		"Virtual Connection not assigned")

	ErrChannelAlreadyClosing = NewError(
		"Channel already closing")
)

// Consts
const (
	maxDepleteBufSize = 256
)

// Channelizer represents a Channel Connection Manager
type Channelizer interface {
	Dispatch(ch.Channels) (ch.ID, fsm.FSM, error)
	Timeout(time.Duration)
	For(ch.ID) Virtual
	Shutdown() error
	Closed() <-chan struct{}
}

// Virtual means represents a Virtual Channel
type Virtual interface {
	rw.ReadWriteDepleteDoner

	Timeout(time.Duration)
	CloseAll() error
}

// channelWriteBuf creates and manages Channel Write Buffer
type channelWriteBuf struct {
	buf []byte
}

// Fetch returns a buf for Channel to write, create one if
// needed.
func (c *channelWriteBuf) Fetch(size int) []byte {
	if len(c.buf) >= size {
		return c.buf[:size]
	}

	if size%2 != 0 {
		c.buf = make([]byte, size+1)
	} else {
		c.buf = make([]byte, size)
	}

	return c.buf
}

// channelize implements Channelizer
type channelize struct {
	conn              network.Connection
	timeout           time.Duration
	timeoutTicker     <-chan time.Time
	buf               [3]byte
	channels          [ch.MaxChannels]*channel
	channelsLock      sync.Mutex
	dispatchCompleted chan struct{}
	downSignal        chan struct{}
	downIgniter       chan struct{}
	downNotify        chan struct{}
	writeBuffer       channelWriteBuf
	writeLock         sync.Mutex
}

// channelReader is the dispatched channel reader data
type channelReader struct {
	Connection network.Connection
	Length     uint16
	Complete   chan struct{}
}

// channel implements Channel
type channel struct {
	network.Connection

	id            ch.ID
	parent        Channelizer
	timeout       time.Duration
	timeoutTicker <-chan time.Time
	connClosed    <-chan struct{}
	connReader    chan channelReader
	currentReader channelReader
	downSignal    chan struct{}
	writeBuffer   *channelWriteBuf
	writeLock     *sync.Mutex
}

// Channelize creates a Connection Channel for mulit-channel dispatch
func Channelize(
	c network.Connection,
	timeout time.Duration,
	timeoutTicker <-chan time.Time,
) Channelizer {
	return &channelize{
		conn:              errorconn{Connection: c},
		timeout:           timeout,
		timeoutTicker:     timeoutTicker,
		buf:               [3]byte{},
		channels:          [ch.MaxChannels]*channel{},
		channelsLock:      sync.Mutex{},
		dispatchCompleted: make(chan struct{}, 1),
		downSignal:        make(chan struct{}, 1),
		downNotify:        make(chan struct{}),
		downIgniter:       make(chan struct{}, 1),
		writeBuffer:       channelWriteBuf{},
		writeLock:         sync.Mutex{},
	}
}

// Closed returns a channel that will be closed when current Channelizer
// is down
func (c *channelize) Closed() <-chan struct{} {
	return c.downNotify
}

// Timeout set the read timeout
func (c *channelize) Timeout(t time.Duration) {
	c.timeout = t

	c.conn.SetTimeout(t)
}

// Initialize reads initialization data from Connection
func (c *channelize) Dispatch(channels ch.Channels) (ch.ID, fsm.FSM, error) {
	// Write will be blocked until someone released the lock
	select {
	case c.dispatchCompleted <- struct{}{}:

	case <-c.conn.Closed():
		if c.dispatchCompleted != nil {
			close(c.dispatchCompleted)

			c.dispatchCompleted = nil
		}

		return 0, nil, ErrChannelConnectionDropped

	case d := <-c.downSignal: // Unblock dispatchCompleted with a Channel
		// Forget about dispatchCompleted, we're downing
		if c.dispatchCompleted != nil {
			close(c.dispatchCompleted)

			c.dispatchCompleted = nil
		}

		c.downSignal <- d

		return 0, nil, ErrChannelShuttedDown
	}

	_, rErr := io.ReadFull(c.conn, c.buf[:3])

	if rErr != nil {
		<-c.dispatchCompleted

		return 0, nil, rErr
	}

	machine, fsmErr := channels.Get(ch.ID(c.buf[0]))

	if fsmErr != nil {
		<-c.dispatchCompleted

		return 0, nil, fsmErr
	}

	// Deliever the Read Connection to Virtual Channel
	if uint8(c.buf[0]) >= channels.Size() || c.channels[c.buf[0]] == nil {
		<-c.dispatchCompleted

		return 0, nil, ErrChannelDispatchChannelUnavailable
	}

	segDataLen := uint16(0)
	segDataLen |= uint16(c.buf[1])
	segDataLen <<= 8
	segDataLen |= uint16(c.buf[2])

	select {
	case c.channels[c.buf[0]].connReader <- channelReader{
		Connection: c.conn,
		Length:     segDataLen,
		Complete:   c.dispatchCompleted}:
		return ch.ID(c.buf[0]), machine, nil

	case <-c.conn.Closed():
		if c.dispatchCompleted != nil {
			<-c.dispatchCompleted

			close(c.dispatchCompleted)
			c.dispatchCompleted = nil
		}

		return 0, nil, ErrChannelConnectionDropped

	case d := <-c.downSignal:
		if c.dispatchCompleted != nil {
			<-c.dispatchCompleted

			close(c.dispatchCompleted)
			c.dispatchCompleted = nil
		}

		c.downSignal <- d

		return 0, nil, ErrChannelShuttedDown
	}
}

// For creates a Virtual Channel Connection reader for specified Channel
func (c *channelize) For(id ch.ID) Virtual {
	c.channelsLock.Lock()
	defer c.channelsLock.Unlock()

	if c.channels[id] != nil {
		return c.channels[id]
	}

	c.channels[id] = &channel{
		Connection:    c.conn,
		id:            id,
		parent:        c,
		timeout:       c.timeout,
		timeoutTicker: c.timeoutTicker,
		connClosed:    c.conn.Closed(),
		connReader:    make(chan channelReader, 1),
		currentReader: channelReader{
			Connection: nil,
			Complete:   nil,
			Length:     0,
		},
		downSignal:  c.downSignal,
		writeBuffer: &c.writeBuffer,
		writeLock:   &c.writeLock,
	}

	return c.channels[id]
}

// Shutdown closes all underlaying Virtual Channels
func (c *channelize) Shutdown() error {
	c.channelsLock.Lock()
	defer c.channelsLock.Unlock()

	select {
	case c.downIgniter <- struct{}{}:
		downIgniter := c.downIgniter
		c.downIgniter = nil
		<-downIgniter

		close(c.downNotify)
		close(downIgniter)

		c.downSignal <- struct{}{}

	default:
		return ErrChannelShuttedDown
	}

	for cIdx := range c.channels {
		if c.channels[cIdx] == nil {
			continue
		}

		c.channels[cIdx] = nil
	}

	return nil
}

// Timeout set the read timeout
func (c *channel) Timeout(t time.Duration) {
	c.timeout = t
}

// Depleted returns whether or not there are still remaining data
// in the current Virtual Channel to read
func (c *channel) Depleted() bool {
	return c.currentReader.Length <= 0
}

// Deplete ditchs all remaining data of current segment
func (c *channel) Deplete() error {
	if c.Depleted() {
		return nil
	}

	maxBufSize := c.currentReader.Length

	if maxBufSize > maxDepleteBufSize {
		maxBufSize = maxDepleteBufSize
	}

	dBuf := make([]byte, maxBufSize)

	for {
		_, rErr := c.Read(dBuf)

		if rErr != nil {
			return rErr
		}

		if !c.Depleted() {
			continue
		}

		return nil
	}
}

// Done finish use of current Virtual Channel
func (c *channel) Done() error {
	var err error

	if !c.Depleted() {
		err = c.Deplete()
	}

	if c.currentReader.Connection != nil {
		c.currentReader.Connection = nil
		<-c.currentReader.Complete
	}

	return err
}

// Read reads data from Virtual Channel
func (c *channel) Read(b []byte) (int, error) {
	if c.currentReader.Connection == nil {
		timeoutExpire := time.Now().Add(c.timeout)
		timeoutTicker := c.timeoutTicker

		if c.timeout <= 0 {
			timeoutTicker = nil
		}

		hasReader := false

		for !hasReader {
			select {
			case reader := <-c.connReader:
				c.currentReader.Connection = reader.Connection
				c.currentReader.Length = reader.Length
				c.currentReader.Complete = reader.Complete

				hasReader = true

			case <-c.connClosed:
				return 0, ErrChannelConnectionDropped

			case d := <-c.downSignal:
				c.downSignal <- d

				return 0, ErrChannelShuttedDown

			case <-timeoutTicker:
				if time.Now().Before(timeoutExpire) {
					continue
				}

				return 0, ErrChannelVirtualConnectionTimedout
			}
		}
	} else if c.Depleted() {
		return 0, ErrChannelVirtualConnectionSegmentDepleted
	}

	bufLen := len(b)
	maxReadLen := int(c.currentReader.Length)

	if maxReadLen > bufLen {
		maxReadLen = bufLen
	}

	rLen, rErr := c.currentReader.Connection.Read(b[:maxReadLen])

	c.currentReader.Length -= uint16(rLen)

	return rLen, rErr
}

// Write writes data to a Connection Channel
func (c *channel) Write(b []byte) (int, error) {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	startPos := 0
	bLen := len(b)

	segLen := bLen

	if segLen > math.MaxUint16 {
		segLen = math.MaxUint16
	}

	writeBuf := c.writeBuffer.Fetch(segLen + 3)

	for bLen > startPos {
		writeBuf[0] = c.id.Byte()
		writeBuf[1] = 0 | byte(segLen>>8)
		writeBuf[2] = 0 | byte(segLen<<8>>8)

		if copy(writeBuf[3:segLen+3], b[startPos:startPos+segLen]) != segLen {
			return startPos, ErrChannelWriteFailedToCopyData
		}

		_, wErr := rw.WriteFull(c.Connection, writeBuf[:segLen+3])

		if wErr != nil {
			return startPos, wErr
		}

		startPos += segLen
		segLen = bLen - startPos

		if segLen > math.MaxUint16 {
			segLen = math.MaxUint16
		}
	}

	return startPos, nil
}

// Close closes the base connection that all virtual channels binded on
func (c *channel) CloseAll() error {
	sErr := c.parent.Shutdown()

	if sErr != nil {
		return sErr
	}

	return c.Connection.Close()
}
