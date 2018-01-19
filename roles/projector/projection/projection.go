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

package projection

import (
	"errors"
	"io"
	"time"

	"github.com/reinit/coward/roles/common/network"
)

// Errors
var (
	ErrReceiveNoReceiverAvailable = errors.New(
		"No Receiver is available to receive the Accessor")

	ErrReceiveTimedout = errors.New(
		"Receiver is unable to receive the Accessor in time")

	ErrProccessorExecutionTimedout = errors.New(
		"Proccessor failed to execute the request in given time")

	ErrAccessorClosedBeforeRequestReceive = errors.New(
		"Accessor is closed before request dispatch")
)

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
	requestRetries     uint8
}

// request select a receiver and request relay from a Projection
func (p *projection) request(
	cancel <-chan struct{},
	timeout time.Time,
	req func(r *receiver) (bool, error),
) (bool, error) {
	var selectedReceiver *receiver

	for {
		select {
		case <-p.receivers.Capcity: // Do we had receivers available?
			p.receivers.Capcity <- struct{}{}

		case <-cancel:
			return false, io.EOF

		case <-p.receiveTimeoutTick: // Timeout tick
			now := time.Now()

			if now.Before(timeout) {
				continue
			}

			return false, ErrReceiveNoReceiverAvailable
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

	return req(selectedReceiver)
}

// receive sends request to a receiver and execute the request
func (p *projection) receive(
	cancel <-chan struct{},
	timeout time.Time,
	req func(r *receiver) (chan struct{}, bool, error),
) error {
	var retryWaiter chan struct{}
	var lastErr error

	for retried := uint8(0); retried < p.requestRetries; retried++ {
		retryWaiter = nil

		eR, eE := p.request(cancel, timeout, func(r *receiver) (bool, error) {
			reqRetryWaiter, reqRetry, reqErr := req(r)

			if reqErr == nil {
				return false, nil
			}

			if !reqRetry {
				return false, reqErr
			}

			retryWaiter = reqRetryWaiter

			return true, reqErr
		})

		if retryWaiter != nil {
			<-retryWaiter
		}

		lastErr = eE

		if eE == nil {
			return nil
		}

		if eR {
			continue
		}

		return eE
	}

	return lastErr
}

// Receive deliver a request to one receiver
func (p *projection) Receive(c network.Connection) error {
	exp := time.Now().Add(p.requestTimeout)

	runn := runner{
		callReceive: make(chan runerCall),
	}

	defer runn.Close()

	acc := accessor{
		access:     c,
		proccessor: make(chan Proccessor),
		result:     make(chan accessorResult, 1),
		runner:     runn,
	}

	defer func() {
		close(acc.proccessor)
		close(acc.result)

		for range acc.result {
			// Do nothing but ditch
		}
	}()

	return p.receive(
		c.Closed(), exp, func(r *receiver) (chan struct{}, bool, error) {
			for {
				select {
				case r.accessorChan <- acc:
					proccessor := <-acc.proccessor

					for {
						select {
						case calling := <-runn.callReceive:
							calling.Result <- calling.Job(calling.Logger)

							result := <-acc.result

							// If request been relayed, don't retry regardless
							// whether or not we have failed.
							if result.err == nil {
								return result.wait, false, nil
							}

							return result.wait, false, result.err

						case result := <-acc.result:
							if result.err == nil {
								return result.wait, false, nil
							}

							if result.resetProccessor {
								proccessor.Proccessor.Close()

								return proccessor.Waiter,
									result.retriable, result.err
							}

							return result.wait, result.retriable, result.err

						case <-p.receiveTimeoutTick:
							now := time.Now()

							if now.Before(exp) {
								continue
							}

							// Close Proccessor, it should disconnect the
							// Proccessor from Projector, after that, we can
							// try the next client or tell client that we have
							// failed.
							proccessor.Proccessor.Close()

							// Ditch the result
							<-acc.result

							return proccessor.Waiter, true,
								ErrProccessorExecutionTimedout
						}
					}

				case <-c.Closed():
					return nil, false, ErrAccessorClosedBeforeRequestReceive

				case <-p.receiveTimeoutTick:
					now := time.Now()

					if now.Before(exp) {
						continue
					}

					return nil, false, ErrReceiveTimedout
				}
			}
		})
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
