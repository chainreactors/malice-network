package core

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTickerCallbackFires(t *testing.T) {
	withIsolatedTicker(t)

	var count atomic.Int32
	_, err := GlobalTicker.Start(1, func() {
		count.Add(1)
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	deadline := time.After(3 * time.Second)
	for {
		if count.Load() > 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatal("ticker callback did not fire within 3 seconds")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestTickerRemoveStopsCallback(t *testing.T) {
	withIsolatedTicker(t)

	var count atomic.Int32
	id, err := GlobalTicker.Start(1, func() {
		count.Add(1)
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for at least one tick.
	time.Sleep(1500 * time.Millisecond)
	GlobalTicker.Remove(id)
	snapshot := count.Load()

	// Wait another interval and verify no new ticks.
	time.Sleep(1500 * time.Millisecond)
	if count.Load() > snapshot+1 {
		t.Fatalf("callback fired %d times after Remove (snapshot was %d)", count.Load(), snapshot)
	}
}

func TestTickerRemoveAllStopsAll(t *testing.T) {
	withIsolatedTicker(t)

	var count atomic.Int32
	for i := 0; i < 3; i++ {
		_, err := GlobalTicker.Start(1, func() {
			count.Add(1)
		})
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}
	}

	time.Sleep(1500 * time.Millisecond)
	GlobalTicker.RemoveAll()
	snapshot := count.Load()

	time.Sleep(1500 * time.Millisecond)
	if count.Load() > snapshot {
		t.Fatalf("callbacks still firing after RemoveAll: before=%d after=%d", snapshot, count.Load())
	}
}
