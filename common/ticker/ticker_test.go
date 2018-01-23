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
	"sync"
	"testing"
	"time"
)

func TestTickerBootClose(t *testing.T) {
	tt := New(1*time.Second, 1024)

	ttt, tttErr := tt.Serve()

	if tttErr != nil {
		t.Error("Failed to boot up Ticker due to error:", tttErr)

		return
	}

	_, tttErr = tt.Serve()

	if tttErr == nil {
		t.Error("Start a started Ticker should cause error")

		return
	}

	tttCloseErr := ttt.Close()

	if tttCloseErr != nil {
		t.Error("Failed to close Ticker due to error:", tttCloseErr)

		return
	}

	for i := 0; i < 1000; i++ {
		ttt, tttErr = tt.Serve()

		if tttErr != nil {
			t.Error("Failed to boot up Ticker due to error:", tttErr)

			return
		}

		tttCloseErr = ttt.Close()

		if tttCloseErr != nil {
			t.Error("Failed to close Ticker due to error:", tttCloseErr)

			return
		}
	}

	tttCloseErr = ttt.Close()

	if tttCloseErr == nil {
		t.Error("Close a closed Ticker should cause error")

		return
	}
}

func TestTickerRequest(t *testing.T) {
	const testItems = 1000

	tt, ttErr := New(300*time.Millisecond, 1024).Serve()

	if ttErr != nil {
		t.Error("Failed to boot up Ticker due to error:", ttErr)

		return
	}

	type testResult struct {
		Took   time.Duration
		Waiter *waiter
	}

	results := make([]testResult, testItems)
	shutdownWait := sync.WaitGroup{}

	for i := 0; i < testItems; i++ {
		shutdownWait.Add(1)

		go func(rIdx int) {
			defer shutdownWait.Done()

			start := time.Now()

			rq, rqErr := tt.Request(start.Add(1 * time.Second))

			if rqErr != nil {
				t.Error("Failed to send request due to error:", rqErr)

				return
			}

			<-rq.Wait()

			results[rIdx] = testResult{
				Took:   time.Now().Sub(start),
				Waiter: rq.(*waiter),
			}
		}(i)
	}

	shutdownWait.Wait()

	tt.Close()

	for i := range results {
		if results[i].Took < 1*time.Second {
			t.Error("Ticker must not be ticked before timeout (1 second) "+
				"has reached. Request %d failed to do that, it ticked at %s",
				i, results[i])

			return
		}

		if results[i].Waiter.Next != nil {
			t.Error("Ticked waiter should not point it's next waiter, "+
				"waiter %d didn't do that", i)

			return
		}

		if results[i].Waiter.Previous != nil {
			t.Error("Ticked waiter should not point it's previous waiter, "+
				"waiter %d didn't do that", i)

			return
		}

		if !results[i].Waiter.Disabled {
			t.Error("Ticked waiter should be marked as disabled, "+
				"waiter %d didn't do that", i)

			return
		}
	}
}
