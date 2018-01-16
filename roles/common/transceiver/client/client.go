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
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/reinit/coward/common/fsm"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/timer"
	"github.com/reinit/coward/roles/common/channel"
	"github.com/reinit/coward/roles/common/network"
	"github.com/reinit/coward/roles/common/transceiver"
	"github.com/reinit/coward/roles/common/transceiver/connection"
)

const (
	// channel.MaxChannels are actually an uint8 type, so it's
	// safe to cast it to uint32
	maxChannels = uint32(channel.MaxChannels)
)

// Errors
var (
	ErrConnectionChannelShuttedDownUnexpectedly = errors.New(
		"Selected Connection Channel has been shutted down unexpectedly")

	ErrConnectionConnectCanceled = errors.New(
		"Connect request has been canceled")

	ErrConnectionConnectFailed = errors.New(
		"Failed to connect to the remote")

	ErrConnectionShutdownClosing = errors.New(
		"Connection is closing for shutdown")

	ErrRequestSelectedChannelIsUnavailable = errors.New(
		"Selected Connection Channel is unavailable")

	ErrNotReadyToRequest = errors.New(
		"Client is not ready to handle requests")

	ErrAlreadyServing = errors.New(
		"Client already serving")

	ErrAlreadyClosed = errors.New(
		"Client already closed")
)

// virtualChannel is the data of a Virtual Channel
type virtualChannel struct {
	ID             transceiver.ConnectionID
	ChannelID      channel.ID
	Channel        connection.Virtual
	Closed         <-chan struct{}
	ConnClosed     <-chan struct{}
	Requesting     chan uint32
	IdleTimeout    time.Duration
	InitialTimeout time.Duration
}

// connectedConnection is the data of a Connected Connection
type connectedConnection struct {
	ID         transceiver.ConnectionID
	Connection network.Connection
	Closed     bool
}

// connectRequest contains the Connect Request
type connectRequest struct {
	Exit   bool
	Timer  timer.Stopper
	Result chan connectRequestResult
}

// connectRequestResult is the result of connectRequest
type connectRequestResult struct {
	ID    transceiver.ConnectionID
	Error error
}

// dialer is the Client Connection Dialer
type dialer struct {
	Dialer                   network.Dialer
	InitialTimeout           time.Duration
	IdleTimeout              time.Duration
	MaxConcurrentConnections uint32
	MaxConnectionChannels    uint8
	ConnectionPersistent     bool
}

// dialers is a group of dialer
type dialers []dialer

// TotalConcurrentConnections returns the Total Concurrent Connections count
func (d dialers) TotalConcurrentConnections() uint32 {
	result := uint32(0)

	for _, c := range d {
		result += c.MaxConcurrentConnections
	}

	return result
}

// client implements transceiver.Client
type client struct {
	id                   transceiver.ClientID
	log                  logger.Logger
	dialers              []dialer
	codec                transceiver.CodecBuilder
	cfg                  Config
	bootLock             sync.Mutex
	booted               bool
	running              chan struct{}
	channel              chan virtualChannel
	totalChannels        uint32
	connectionConnect    chan connectRequest
	connectionConnected  chan connectedConnection
	connectionFree       chan struct{}
	connectionWait       sync.WaitGroup
	connectionWorkers    uint32
	connectionEnabled    chan struct{}
	connectionClosing    chan struct{}
	connectionCloserLock sync.Mutex
	lastConnectionID     transceiver.ConnectionID
	requestRetries       uint8
	requestWaitTicker    <-chan time.Time
}

// New creates a new transceiver.Client
func New(
	clientID transceiver.ClientID,
	log logger.Logger,
	d network.Dialer,
	codec transceiver.CodecBuilder,
	requestWaitTicker <-chan time.Time,
	cfg Config,
) transceiver.Client {
	dls := dialers{
		dialer{
			Dialer:                   d,
			InitialTimeout:           cfg.InitialTimeout,
			IdleTimeout:              cfg.IdleTimeout,
			MaxConcurrentConnections: cfg.MaxConcurrent,
			MaxConnectionChannels:    cfg.ConnectionChannels,
			ConnectionPersistent:     cfg.ConnectionPersistent,
		},
	}

	return &client{
		id: clientID,
		log: log.Context("Transceiver (" +
			strconv.FormatUint(uint64(clientID), 10) + ")"),
		dialers:  dls,
		codec:    codec,
		cfg:      cfg,
		bootLock: sync.Mutex{},
		booted:   false,
		running:  make(chan struct{}, cfg.MaxConcurrent),
		channel: make(chan virtualChannel,
			dls.TotalConcurrentConnections()*maxChannels),
		totalChannels:        0,
		connectionConnect:    make(chan connectRequest),
		connectionConnected:  make(chan connectedConnection, cfg.MaxConcurrent),
		connectionFree:       make(chan struct{}),
		connectionWait:       sync.WaitGroup{},
		connectionWorkers:    0,
		connectionEnabled:    make(chan struct{}, 1),
		connectionClosing:    nil,
		connectionCloserLock: sync.Mutex{},
		lastConnectionID:     0,
		requestRetries:       cfg.RequestRetries,
		requestWaitTicker:    requestWaitTicker,
	}
}

// connect connects remote host
func (c *client) connect(
	connectionID transceiver.ConnectionID,
	dial network.Dial,
	d dialer,
	l logger.Logger,
	tm timer.Stopper,
	result chan connectRequestResult,
) error {
	log := l.Context(dial.String())
	needClose := true

	// Dial
	conn, dialErr := dial.Dial()

	if dialErr != nil {
		result <- connectRequestResult{
			ID:    connectionID,
			Error: ErrConnectionConnectFailed,
		}

		return dialErr
	}

	tm.Stop()

	defer func() {
		log.Debugf("Connection lost")

		if !needClose {
			return
		}

		conn.Close()
	}()

	log = log.Context(conn.LocalAddr().String())

	// Publish connection
	select {
	case c.connectionConnected <- connectedConnection{
		ID:         connectionID,
		Connection: conn,
		Closed:     false,
	}:

	case <-c.connectionClosing:
		return ErrConnectionShutdownClosing
	}

	defer func() {
		c.connectionCloserLock.Lock()
		defer c.connectionCloserLock.Unlock()

		for cc := range c.connectionConnected {
			if cc.ID != connectionID {
				c.connectionConnected <- cc

				continue
			}

			needClose = !cc.Closed

			break
		}
	}()

	// Test again even when connectionConnected writing has finished
	select {
	case <-c.connectionClosing:
		return ErrConnectionShutdownClosing

	case c.connectionEnabled <- struct{}{}:
		<-c.connectionEnabled
	}

	// Init connection
	channelCreated := 0

	cc, ccErr := connection.Codec(conn, c.codec)

	if ccErr != nil {
		result <- connectRequestResult{
			ID:    connectionID,
			Error: ccErr,
		}

		return ccErr
	}

	channelized := connection.Channelize(
		cc, d.InitialTimeout, c.requestWaitTicker)

	connRequestingCount := make(chan uint32, 1)
	connRequestingCount <- 0

	vChannels := channel.New(func(id channel.ID) fsm.Machine {
		channelCreated++

		vChannel := virtualChannel{
			ID:             connectionID,
			ChannelID:      id,
			Channel:        channelized.For(id),
			Closed:         channelized.Closed(),
			ConnClosed:     conn.Closed(),
			Requesting:     connRequestingCount,
			IdleTimeout:    d.IdleTimeout,
			InitialTimeout: d.InitialTimeout,
		}

		c.channel <- vChannel

		return handler{}
	}, d.MaxConnectionChannels)

	defer func() {
		// Shutdown all Channels and Channelized connection before
		// clean up Virtual Connection from c.channel. Otherwise,
		// we may get stuck during Virtual Connection release below.
		vChannels.Shutdown()
		channelized.Shutdown()

		c.connectionCloserLock.Lock()
		defer c.connectionCloserLock.Unlock()

		for ch := range c.channel {
			if ch.ID != connectionID {
				c.channel <- ch

				continue
			}

			channelCreated--

			if channelCreated > 0 {
				continue
			}

			break
		}
	}()

	bootUpErr := vChannels.All(func(id channel.ID, m fsm.FSM) (bool, error) {
		return true, m.Bootup()
	})

	if bootUpErr != nil {
		result <- connectRequestResult{
			ID:    connectionID,
			Error: bootUpErr,
		}

		return bootUpErr
	}

	result <- connectRequestResult{
		ID:    connectionID,
		Error: nil,
	}

	// Serve
	connectionTimeoutUpdated := false

	log.Debugf("Ready")

	for {
		cID, machine, chGetErr := channelized.Dispatch(vChannels)

		if chGetErr != nil {
			return chGetErr
		}

		if !connectionTimeoutUpdated {
			channelized.Timeout(d.IdleTimeout)

			connectionTimeoutUpdated = true
		}

		if !machine.Running() {
			return ErrConnectionChannelShuttedDownUnexpectedly
		}

		tickErr := machine.Tick()

		if tickErr == nil {
			continue
		}

		log.Debugf("An error occured during request handling for "+
			"Channel %d: %s", cID, tickErr)

		return tickErr
	}
}

// connection handle and maintains connection
func (c *client) connection(
	id transceiver.ConnectionID,
	d dialer,
	ready chan struct{},
) {
	log := c.log.Context(
		"Connection (" + strconv.FormatUint(uint64(id), 10) + ")")

	c.running <- struct{}{}

	defer func() {
		<-c.running

		log.Debugf("Closed")

		c.connectionWait.Done()
	}()

	dial := d.Dialer.Dialer()

	for {
		select {
		case ready <- struct{}{}:
			ready = nil // Replace local value to nil

		case <-c.connectionFree:
			// Do nothing

		case req := <-c.connectionConnect:
			if req.Exit {
				req.Result <- connectRequestResult{
					ID:    id,
					Error: nil,
				}

				return
			}

			log.Debugf("Connecting")

			connectorErr := c.connect(id, dial, d, log, req.Timer, req.Result)

			if connectorErr == nil {
				log.Debugf("Connection to \"%s\" is lost", dial)

				continue
			}

			log.Debugf("Connection to \"%s\" is lost: %s",
				dial, connectorErr)
		}
	}
}

// getConnection returns an Virtual Channel
func (c *client) getConnection(
	cancel <-chan struct{},
	tm timer.Stopper,
) (virtualChannel, bool, error) {
	// First try: See if there're some free channels
	select {
	case cc := <-c.channel:
		return cc, true, nil

	case c.running <- struct{}{}:
		<-c.running

		return virtualChannel{}, true, ErrNotReadyToRequest

	case <-c.connectionClosing:
		return virtualChannel{}, false, ErrConnectionShutdownClosing

	case <-cancel:
		return virtualChannel{}, false, ErrConnectionConnectCanceled

	default:
	}

	// No? OK, ask link to be established
	connectReq := connectRequest{
		Exit:   false,
		Timer:  tm,
		Result: make(chan connectRequestResult),
	}
	connectionConnect := c.connectionConnect
	retryRemains := c.requestRetries

	for {
		select {
		case c.running <- struct{}{}:
			<-c.running

			return virtualChannel{}, true, ErrNotReadyToRequest

		case <-c.connectionClosing:
			return virtualChannel{}, false, ErrConnectionShutdownClosing

		case connectionConnect <- connectReq:
			rr := <-connectReq.Result

			if rr.Error != nil {
				retryRemains--

				if retryRemains > 0 {
					continue
				} else {
					return virtualChannel{}, true, rr.Error
				}
			}

			// If one connection is established, don't connect another
			connectionConnect = nil

		case cc := <-c.channel:
			return cc, true, nil

		case <-cancel:
			return virtualChannel{}, false, ErrConnectionConnectCanceled
		}
	}
}

// Serve start serving
func (c *client) Serve() (transceiver.Requester, error) {
	c.bootLock.Lock()
	defer c.bootLock.Unlock()

	if c.booted {
		return nil, ErrAlreadyServing
	}

	ready := make(chan struct{})
	defer close(ready)

	// Start all connectors and get them ready
	c.lastConnectionID = 0
	c.totalChannels = 0
	c.connectionClosing = make(chan struct{})

	for _, dialer := range c.dialers {
		for cIdx := uint32(0); cIdx < dialer.MaxConcurrentConnections; cIdx++ {
			c.connectionWait.Add(1)

			go c.connection(
				c.lastConnectionID,
				dialer,
				ready,
			)

			<-ready

			c.totalChannels += uint32(dialer.MaxConnectionChannels)
			c.lastConnectionID++
			c.connectionWorkers++
		}
	}

	c.booted = true

	return c, nil
}

// Close stop serving
func (c *client) Close() error {
	c.bootLock.Lock()
	defer c.bootLock.Unlock()

	if !c.booted {
		return ErrAlreadyClosed
	}

	c.log.Debugf("Closing")

	c.connectionEnabled <- struct{}{}

	defer func() {
		<-c.connectionEnabled
	}()

	close(c.connectionClosing)

	// connection down
	connectExitReq := connectRequest{
		Exit:   true,
		Timer:  nil,
		Result: make(chan connectRequestResult),
	}

	connectionConnected := c.connectionConnected
	remainingWorkersToClose := c.connectionWorkers

	for c.connectionWorkers > 0 {
		select {
		case cc := <-connectionConnected:
			if cc.Closed {
				connectionConnected <- cc

				continue
			}

			cc.Connection.Close()
			cc.Closed = true

			connectionConnected <- cc

			remainingWorkersToClose--

			if remainingWorkersToClose > 0 {
				continue
			}

			connectionConnected = nil

		case c.connectionConnect <- connectExitReq:
			<-connectExitReq.Result

			c.connectionWorkers--
		}
	}

	c.log.Debugf("Waiting for all connection handler to quit")

	c.connectionWait.Wait()

	c.booted = false

	c.log.Debugf("Closed")

	return nil
}

// ID returns ID of current Client
func (c *client) ID() transceiver.ClientID {
	return c.id
}

// Connections returns how many connections can be established by current
// client
func (c *client) Connections() uint32 {
	return c.connectionWorkers
}

// Channels returns the total Channel count that can be established by this
// client
func (c *client) Channels() uint32 {
	return c.totalChannels
}

// Full returns whether or not the Client is fully connected, that means
// max amount of connection is established with the remote server
func (c *client) Full() bool {
	select {
	case c.connectionFree <- struct{}{}:
		return false

	default:
		return true
	}
}

// Available check if there are free Channels for request
func (c *client) Available() bool {
	select {
	case cc := <-c.channel:
		c.channel <- cc

		return true

	default:
		return false
	}
}

func (c *client) request(
	requestBuilder transceiver.RequestBuilder,
	cancel <-chan struct{},
	meter transceiver.Meter,
	log logger.Logger,
) (bool, bool, logger.Logger, error) {
	ch, chRetriable, chErr := c.getConnection(cancel, meter.Connection())

	if chErr != nil {
		meter.ConnectionFailure(chErr)

		return chRetriable, false, log, chErr
	}

	defer func() {
		c.channel <- ch
	}()

	connLogger := log.Context("Connection (" +
		strconv.FormatUint(uint64(ch.ID), 10) + ")")

	meter.ConnectionFailure(nil)

	connRequesting := <-ch.Requesting
	ch.Requesting <- connRequesting + 1

	defer func() {
		endConnRequesting := <-ch.Requesting - 1

		if !c.cfg.ConnectionPersistent && endConnRequesting <= 0 {
			ch.Channel.CloseAll()
		}

		ch.Requesting <- endConnRequesting
	}()

	select {
	case <-ch.Closed:
		return true, true, connLogger, ErrRequestSelectedChannelIsUnavailable

	case <-ch.ConnClosed:
		return true, true, connLogger, ErrRequestSelectedChannelIsUnavailable

	default:
	}

	connLogger = connLogger.Context("Channel (" +
		strconv.FormatUint(uint64(ch.ChannelID), 10) + ")")

	reqTimer := meter.Request()

	reqFSM := fsm.New(requestBuilder(ch.ID, ch.Channel, connLogger))

	ch.Channel.Timeout(ch.InitialTimeout)

	initErr := reqFSM.Bootup()

	if initErr != nil {
		_, connErr := initErr.(connection.Error)

		if connErr {
			ch.Channel.CloseAll()
		}

		return true, false, connLogger, initErr
	}

	defer reqFSM.Shutdown()

	reqTimer.Stop()

	ch.Channel.Timeout(ch.IdleTimeout)

	for reqFSM.Running() {
		handleErr := reqFSM.Tick()

		if handleErr == nil {
			continue
		}

		_, connErr := handleErr.(connection.Error)

		if connErr {
			ch.Channel.CloseAll()
		}

		return false, false, connLogger, handleErr
	}

	return false, false, connLogger, nil
}

// Request sends request
func (c *client) Request(
	log logger.Logger,
	requestBuilder transceiver.RequestBuilder,
	cancel <-chan struct{},
	meter transceiver.Meter,
) (bool, error) {
	var retriable bool
	var wontCount bool
	var connLog logger.Logger
	var err error

	log = log.Context("Transceiver (" +
		strconv.FormatUint(uint64(c.id), 10) + ")")

	for retried := uint8(0); retried < c.requestRetries; retried++ {
		retriable, wontCount, connLog, err = c.request(
			requestBuilder, cancel, meter, log)

		if err == nil {
			break
		}

		if wontCount {
			retried--

			continue
		}

		if !retriable {
			connLog.Debugf("Request has failed due to error: %s. Given up",
				err)

			break
		}

		connLog.Debugf("Request has failed due to error: %s. Retrying (%d/%d)",
			err, retried+1, c.requestRetries)
	}

	return retriable, err
}
