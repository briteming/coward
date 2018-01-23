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

package ticker

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestWaitersInsert(t *testing.T) {
	ws := waiters{
		Head: nil,
		Tail: nil,
	}

	testNow := time.Now()

	ws.Insert(1*time.Second, testNow)
	ws.Insert(1*time.Second, testNow.Add(10*time.Second))

	ws.Insert(1*time.Second, testNow)
	ws.Insert(1*time.Second, testNow.Add(10*time.Second))

	ws.Insert(1*time.Second, testNow.Add(-1*time.Second))
	ws.Insert(1*time.Second, testNow.Add(11*time.Second))

	ws.Insert(1*time.Second, testNow.Add(2*time.Second))
	ws.Insert(1*time.Second, testNow.Add(3*time.Second))

	ws.Insert(1*time.Second, testNow.Add(7*time.Second))
	ws.Insert(1*time.Second, testNow.Add(8*time.Second))

	oldNum := int64(0)

	ws.Retrieve(func(ww *waiter) bool {
		if oldNum >= ww.When {
			t.Error("Waiters must be sorted during insertion and " +
				"be retrieve in ASC order")

			return false
		}

		oldNum = ww.When

		return true
	})
}

func TestWaitersInsert2(t *testing.T) {
	ws := waiters{
		Head: nil,
		Tail: nil,
	}

	for i := 0; i < 1000; i++ {
		ws.Insert(1*time.Second, time.Unix(rand.Int63(), 0))
	}

	oldNum := int64(math.MinInt64)

	ws.Retrieve(func(ww *waiter) bool {
		if oldNum >= ww.When {
			t.Error("Waiters must be sorted during insertion and " +
				"be retrieve in ASC order")

			return false
		}

		oldNum = ww.When

		return true
	})
}

func TestWaitersInsert3(t *testing.T) {
	ws := waiters{
		Head: nil,
		Tail: nil,
	}

	now := time.Now()

	w1 := ws.Insert(1*time.Second, now)
	w2 := ws.Insert(1*time.Second, now.Add(400*time.Second))
	w3 := ws.Insert(1*time.Second, now.Add(199*time.Second))

	if ws.Head != w1 || ws.Tail != w2 {
		t.Error("waiters must update head and tail according to new insertion")

		return
	}

	w2.delete()

	if ws.Head != w1 || ws.Tail != w3 {
		t.Error("waiters must update head and tail according to new deletion")

		return
	}
}

func BenchmarkWaitersInsert1000Times(b *testing.B) {
	const benchBatch = 1000

	randomNumPool := [benchBatch]time.Time{}

	for i := 0; i < benchBatch; i++ {
		randomNumPool[i] = time.Unix(rand.Int63(), 0)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for t := 0; t < b.N; t++ {
		ws := waiters{
			Head: nil,
			Tail: nil,
		}

		for i := 0; i < benchBatch; i++ {
			ws.Insert(1*time.Second, randomNumPool[i])
		}
	}
}

func TestWaitersInsertAndDelete(t *testing.T) {
	ws := waiters{
		Head: nil,
		Tail: nil,
	}

	testNow := time.Now()

	wt := ws.Insert(1*time.Second, testNow)

	wt.delete()

	if ws.Head != nil || ws.Tail != nil {
		t.Error("An empty waiters chain should not have Head and Tail")

		return
	}
}

func TestWaitersUnlinkBy(t *testing.T) {
	const testItems = 100

	ws := waiters{
		Head: nil,
		Tail: nil,
	}

	wsItem := make([]*waiter, testItems)

	testNow := time.Now()

	for i := time.Duration(0); i < testItems; i++ {
		wsItem[i] = ws.Insert(1*time.Second, testNow.Add(i*time.Second))
	}

	ws.UnlinkBy(wsItem[(testItems/2)-1])

	if ws.Head != wsItem[testItems/2] || ws.Tail != wsItem[testItems-1] {
		t.Error("UnlinkBy must update waiters head and tail after complete")

		return
	}

	itemsLeft := uint64(0)

	ws.Retrieve(func(ww *waiter) bool {
		itemsLeft++

		return true
	})

	if itemsLeft != testItems/2 {
		t.Error("There should be %d items left since we deleted %d, "+
			"but we got %d instead.",
			testItems/2, testItems-(testItems/2), itemsLeft)

		return
	}

	for i := range wsItem[testItems/2:] {
		if wsItem[i].Next != nil {
			t.Error("Deleted waiter should not point it's next waiter, "+
				"waiter %d didn't do that", i)

			return
		}

		if wsItem[i].Previous != nil {
			t.Error("Deleted waiter should not point it's previous waiter, "+
				"waiter %d didn't do that", i)

			return
		}

		if !wsItem[i].Disabled {
			t.Error("Deleted waiter should be marked as disabled, "+
				"waiter %d didn't do that", i)

			return
		}
	}

	ws.UnlinkBy(wsItem[testItems-1])

	if ws.Head != nil || ws.Tail != nil {
		t.Error("UnlinkBy must update waiters head and tail after complete")

		return
	}

	itemsLeft = 0

	ws.Retrieve(func(ww *waiter) bool {
		itemsLeft++

		return true
	})

	if itemsLeft != 0 {
		t.Error("We deleted everything, so there should be no item on the "+
			"chain but we got %d instead.",
			itemsLeft)

		return
	}

	for i := range wsItem {
		if wsItem[i].Next != nil {
			t.Error("Deleted waiter should not point it's next waiter, "+
				"waiter %d didn't do that", i)

			return
		}

		if wsItem[i].Previous != nil {
			t.Error("Deleted waiter should not point it's previous waiter, "+
				"waiter %d didn't do that", i)

			return
		}

		if !wsItem[i].Disabled {
			t.Error("Deleted waiter should be marked as disabled, "+
				"waiter %d didn't do that", i)

			return
		}
	}

	if ws.Head != nil || ws.Tail != nil {
		t.Error("An empty waiters chain should not have Head and Tail")

		return
	}
}
