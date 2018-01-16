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

package fsm

import "testing"

type dummyMachine struct {
	Result int
}

func (d *dummyMachine) StateOne(f FSM) error {
	if d.Result >= 10000 {
		f.Shutdown()

		return nil
	}

	d.Result *= 10

	f.Switch(d.StateTwo)

	return nil
}

func (d *dummyMachine) StateTwo(f FSM) error {
	d.Result++

	f.SwitchTick(d.StateOne)

	return nil
}

func (d *dummyMachine) Bootup() (State, error) {
	return d.StateOne, nil
}

func (d *dummyMachine) Shutdown() error {
	return nil
}

func TestFSM(t *testing.T) {
	d := &dummyMachine{
		Result: 0,
	}
	m := New(d)

	bootupErr := m.Bootup()

	if bootupErr != nil {
		t.Error("Failed to Bootup:", bootupErr)

		return
	}

	for {
		if !m.Running() {
			break
		}

		tickErr := m.Tick()

		if tickErr != nil {
			t.Error("Failed to Tick:", tickErr)

			return
		}
	}

	if d.Result != 11111 {
		t.Error("Expecting result will be 11111, got:", d.Result)

		return
	}
}

func BenchmarkFSMSwitchAndTick(b *testing.B) {
	d := &dummyMachine{
		Result: 0,
	}
	m := New(d)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if !m.Running() {
			d.Result = 0

			m.Bootup()
		}

		m.Tick()
	}
}
