package key

import (
	"bytes"
	"testing"
	"time"
)

func TestTimed(t *testing.T) {
	testTime := time.Time{}
	tt := Timed([]byte("Hello World"), 1*time.Second, func() time.Time {
		return testTime
	})

	firstKey, _ := tt.Get(32)

	resultKey, _ := tt.Get(32)
	if !bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at same time period must remain the same")

		return
	}

	testTime = testTime.Add(500 * time.Millisecond)

	resultKey, _ = tt.Get(32)
	if !bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at same time period must remain the same")

		return
	}

	testTime = testTime.Add(400 * time.Millisecond)

	resultKey, _ = tt.Get(32)
	if !bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at same time period must remain the same")

		return
	}

	testTime = testTime.Add(100 * time.Millisecond)

	resultKey, _ = tt.Get(32)
	if bytes.Equal(firstKey, resultKey) {
		t.Error("Key got at different time period must be different")

		return
	}
}
