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
	"errors"

	"github.com/reinit/coward/roles/projector/projection"
)

// Errors
var (
	ErrRegisterationRemoveReceiverNotFound = errors.New(
		"Cannot delete a non-existing Projectee registeration")
)

// registeration registered data
type registeration struct {
	Receiver projection.Receiver
	Count    int
}

// registerations Projection registeration
type registerations struct {
	projections projection.Projections
	receivers   map[projection.ID]registeration
}

// Register registers or returns a Projection receiver
func (r *registerations) Register(
	id projection.ID) (projection.Receiver, error) {
	receiverData, receiverDataFound := r.receivers[id]

	if receiverDataFound {
		receiverData.Count++
		receiverData.Receiver.Expand()

		r.receivers[id] = receiverData

		return receiverData.Receiver, nil
	}

	proj, projErr := r.projections.Projection(id)

	if projErr != nil {
		return nil, projErr
	}

	newRegisteredProjee := registeration{
		Receiver: proj.Receiver(),
		Count:    1,
	}

	r.receivers[id] = newRegisteredProjee

	return newRegisteredProjee.Receiver, nil
}

// Remove remove or down scale a Projection
func (r *registerations) Remove(id projection.ID) error {
	receiverData, receiverDataFound := r.receivers[id]

	if !receiverDataFound {
		return ErrRegisterationRemoveReceiverNotFound
	}

	receiverData.Count--

	if receiverData.Count > 0 {
		receiverData.Receiver.Shrink()

		r.receivers[id] = receiverData

		return nil
	}

	receiverData.Receiver.Remove()

	delete(r.receivers, id)

	return nil
}

// All iterate through all registered receivers
func (r *registerations) All(item func(receiver projection.Receiver)) {
	for k := range r.receivers {
		item(r.receivers[k].Receiver)
	}
}
