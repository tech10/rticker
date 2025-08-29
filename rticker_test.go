package rticker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tech10/rticker"
)

func TestBasicTicks(t *testing.T) {
	ticker := rticker.New(50 * time.Millisecond)
	defer func() {
		_ = ticker.Close()
	}()

	select {
	case <-ticker.C:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("expected tick, got timeout")
	}
}

func TestResetChangesInterval(t *testing.T) {
	ticker := rticker.New(200 * time.Millisecond)
	defer func() {
		_ = ticker.Close()
	}()

	// Wait for first tick
	select {
	case <-ticker.C:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected tick before reset")
	}

	// Reset to faster interval
	err := ticker.Reset(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error on reset: %v", err)
	}

	// Should tick again quickly
	select {
	case <-ticker.C:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("expected quick tick after reset")
	}
}

func TestStopPreventsTicks(t *testing.T) {
	ticker := rticker.New(50 * time.Millisecond)
	_ = ticker.Stop()

	select {
	case <-ticker.C:
		t.Fatal("unexpected tick after Stop")
	case <-time.After(100 * time.Millisecond):
		// Success: no tick
	}
}

func TestCloseStopsAndCloses(t *testing.T) {
	ticker := rticker.New(30 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		for range ticker.C {
			time.Sleep(time.Millisecond)
		}
		close(done)
	}()

	time.Sleep(70 * time.Millisecond)
	err := ticker.Close()
	if err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ticker channel not closed after Close")
	}
}

func TestCloseError(t *testing.T) {
	ticker := rticker.New(50 * time.Millisecond)
	err := ticker.Close()
	if err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	err = ticker.Close()
	if err == nil {
		t.Fatal("expected error on second Close")
	}
}

func TestResetAfterCloseFails(t *testing.T) {
	ticker := rticker.New(50 * time.Millisecond)
	_ = ticker.Close()
	err := ticker.Reset(10 * time.Millisecond)
	if err == nil {
		t.Fatalf("expected error on Reset after Close")
	}
}

func TestStopAfterCloseIsSafe(t *testing.T) {
	ticker := rticker.New(50 * time.Millisecond)
	_ = ticker.Close()
	err := ticker.Stop()
	if err == nil {
		t.Fatal("expected error when stopping an already closed ticker")
	}
	t.Log("No panic occurred, test successful.")
}

func TestStopMultipleTimesIsSafe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PANIC:\n%v", r)
		}
	}()
	ticker := rticker.New(50 * time.Millisecond)
	err := ticker.Stop()
	if err != nil {
		t.Fatal("expected nil error on first stop")
	}
	err = ticker.Stop()
	if err != nil {
		t.Fatal("expected nil error on second stop")
	}
	t.Log("No panic occurred, test successful.")
}

func TestTickerWithContextCancel(t *testing.T) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	count := 0
	ctx, cancel := context.WithCancel(context.Background())
	ticker := rticker.NewWithContext(ctx, time.Millisecond)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		for range ticker.C {
			t.Log("received 1 tick")
			terminate := false
			mu.Lock()
			count++
			cancel()
			if count > 1 {
				t.Errorf("Count is greater than 1: %d. Manually terminating loop.", count)
				terminate = true
			}
			mu.Unlock()
			if terminate {
				return
			}
		}
	}()
	wg.Wait()
	mu.Lock()
	nc := count
	mu.Unlock()
	if nc != 1 {
		t.Fatalf("Expected 1, got %d", nc)
	}
	t.Log("One tick received, context canceled. No deadlock.")
	if !ticker.IsClosed() {
		t.Fatalf("Expected ticker to be closed.")
	}
	t.Log("Ticker closed successfully.")
}
