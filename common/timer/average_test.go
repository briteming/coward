package timer

import (
	"testing"
	"time"
)

func TestAverage(t *testing.T) {
	a := Average()

	for i := 0; i < 30; i++ {
		aStart := a.Start()

		time.Sleep(100 * time.Millisecond)

		if aStart.Stop() <= 100*time.Millisecond {
			t.Errorf("Expecting an Average time that greater than 100ms, "+
				"got %s", aStart.Stop())

			return
		}
	}
}
