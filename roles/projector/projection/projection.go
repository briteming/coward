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

package projection

import (
	"errors"
	"io"
	"time"

	"github.com/reinit/coward/common/worker"
	"github.com/reinit/coward/roles/common/network"
)

// Errors
var (
	ErrReceiveNoReceiverAvailable = errors.New(
		"No Receiver is available to receive the Accessor")

	ErrReceiveTimedout = errors.New(
		"Receiver is unable to receive the Accessor in time")
)

// Accessor is the accessing requesting client
type Accessor struct {
	Access network.Connection
	Error  chan error
	Runner worker.Runner
}

// Projection is the projection operator
type Projection interface {
	Receive(c network.Connection) error
	Receiver() Receiver
}

// projection implements Projection
type projection struct {
	id                 ID
	receivers          receivers
	receiveTimeoutTick <-chan time.Time
	requestTimeout     time.Duration
}

// Receive deliver a request to one receiver
func (p *projection) Receive(c network.Connection) error {
	var selectedReceiver *receiver

	reqExpireTime := time.Now().Add(p.requestTimeout)

	for {
		select {
		case <-p.receivers.Capcity: // Do we had receivers available?
			p.receivers.Capcity <- struct{}{}

		case <-c.Closed():
			return io.EOF

		case <-p.receiveTimeoutTick: // Timeout tick
			now := time.Now()

			if now.Before(reqExpireTime) {
				continue
			}

			return ErrReceiveNoReceiverAvailable
		}

		p.receivers.Capacitor.L.Lock()

		selectedReceiver = p.receivers.Head

		if selectedReceiver == nil {
			p.receivers.Capacitor.L.Unlock()

			continue
		}

		selectedReceiver.capcity--

		if selectedReceiver.capcity <= 1 {
			selectedReceiver.lifted = true
			selectedReceiver.remove()
		}

		p.receivers.Capacitor.Broadcast()

		p.receivers.Capacitor.L.Unlock()

		break
	}

	defer func() {
		p.receivers.Capacitor.L.Lock()
		defer p.receivers.Capacitor.L.Unlock()

		selectedReceiver.capcity++

		if !selectedReceiver.lifted {
			p.receivers.Capacitor.Broadcast()

			return
		}

		selectedReceiver.lifted = false

		p.receivers.InsertDelayOrder(selectedReceiver)

		p.receivers.Capacitor.Broadcast()
	}()

	runn := runner{
		callReceive: make(chan runerCall),
	}

	defer runn.Close()

	acc := Accessor{
		Access: c,
		Error:  make(chan error),
		Runner: runn,
	}

	for {
		select {
		case selectedReceiver.accessorChan <- acc:
			select {
			case calling := <-runn.callReceive:
				calling.Result <- calling.Job(calling.Logger)

				return <-acc.Error

			case accErr := <-acc.Error:
				return accErr
			}

		case <-c.Closed():
			return io.EOF

		case <-p.receiveTimeoutTick:
			now := time.Now()

			if now.Before(reqExpireTime) {
				continue
			}

			return ErrReceiveTimedout
		}
	}
}

// Receiver creates and registers a new reciever
func (p *projection) Receiver() Receiver {
	p.receivers.Capacitor.L.Lock()
	defer p.receivers.Capacitor.L.Unlock()

	newReceive := &receiver{
		plot:         &p.receivers,
		next:         nil,
		previous:     nil,
		id:           p.id,
		accessorChan: make(chan Accessor),
		delay:        0,
		capcity:      1,
		lifted:       false,
		deleted:      false,
	}

	p.receivers.Insert(newReceive)

	return newReceive
}
