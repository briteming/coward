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

package channel

import (
	"testing"

	"github.com/reinit/coward/common/fsm"
)

type dummyFSMMachine struct {
	d       *[MaxChannels]bool
	current ID
}

func (d *dummyFSMMachine) Run(m fsm.FSM) error {
	d.d[d.current] = true

	return nil
}

func (d *dummyFSMMachine) Bootup() (fsm.State, error) {
	return d.Run, nil
}

func (d *dummyFSMMachine) Shutdown() error {
	d.d[d.current] = false

	return nil
}

func TestChannels(t *testing.T) {
	d := [MaxChannels]bool{}

	c := New(func(id ID) fsm.Machine {
		return &dummyFSMMachine{
			d:       &d,
			current: id,
		}
	}, 255)

	aErr := c.All(func(id ID, m fsm.FSM) (bool, error) {
		if id >= MaxChannels-1 {
			return false, nil
		}

		bErr := m.Bootup()

		if bErr != nil {
			return false, bErr
		}

		tErr := m.Tick()

		if tErr != nil {
			return false, tErr
		}

		return true, nil
	})

	if aErr != nil {
		t.Error("An error happened when handling Channel:", aErr)

		return
	}

	idleChannelID, idleChannel, idleChannelErr := c.Idle()

	if idleChannelErr != nil {
		t.Error("Failed to fetch a idle channel:", idleChannelErr)

		return
	}

	if idleChannelID != MaxChannels-1 {
		t.Errorf("Expecting Idle channel would be %d, got %d",
			MaxChannels-1, idleChannelID)

		return
	}

	bErr := idleChannel.Bootup()

	if bErr != nil {
		t.Error("Failed to bootup idle channel:", bErr)

		return
	}

	tErr := idleChannel.Tick()

	if tErr != nil {
		t.Error("Failed to tick channel:", tErr)

		return
	}

	for k, v := range d {
		if v {
			continue
		}

		t.Errorf("Channel %d failed to set test value", k)

		return
	}

	sErr := c.Shutdown()

	if sErr != nil {
		t.Error("Failed to shutdown Channel due to error:", sErr)

		return
	}

	for k, v := range d {
		if !v {
			continue
		}

		t.Errorf("Channel %d failed to shutdown", k)

		return
	}
}
