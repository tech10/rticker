package rticker_test

import (
	"context"
	"fmt"
	"time"

	"github.com/tech10/rticker"
)

// ExampleNew demonstrates creating and using a basic resettable ticker.
func ExampleNew() {
	t := rticker.New(200 * time.Millisecond)

	count := 0
	for range t.C {
		count++
		fmt.Println("tick", count)
		if count == 3 {
			t.Close()
		}
	}
	fmt.Println("Ticker closed.")

	// Output:
	// tick 1
	// tick 2
	// tick 3
	// Ticker closed.
}

// ExampleNewWithContext demonstrates creating a ticker that is canceled when its context is canceled.
func ExampleNewWithContext() {
	ctx, cancel := context.WithCancel(context.Background())
	t := rticker.NewWithContext(ctx, 30*time.Millisecond)
	defer t.Close()

	// read one tick
	<-t.C

	// cancel the context
	cancel()

	// wait until the ticker loop shuts down
	t.Wait()

	fmt.Println("context canceled, ticker closed:", t.IsClosed())

	// Output:
	// context canceled, ticker closed: true
}

// ExampleT_Reset demonstrates resetting a ticker to a different interval.
func ExampleT_Reset() {
	t := rticker.New(100 * time.Millisecond)
	defer t.Close()

	<-t.C // first tick
	_ = t.Reset(50 * time.Millisecond)

	// We should now get a tick faster than before.
	<-t.C
	fmt.Println("reset worked")

	// Output:
	// reset worked
}

// ExampleT_Stop demonstrates stopping and restarting the ticker.
func ExampleT_Stop() {
	t := rticker.New(50 * time.Millisecond)
	defer t.Close()

	<-t.C
	_ = t.Stop()

	select {
	case <-t.C:
		fmt.Println("unexpected tick")
	case <-time.After(100 * time.Millisecond):
		fmt.Println("stopped successfully")
	}

	// Now restart
	_ = t.Reset(10 * time.Millisecond)
	<-t.C
	fmt.Println("restarted successfully")

	// Output:
	// stopped successfully
	// restarted successfully
}

// ExampleT_Close demonstrates closing the ticker.
func ExampleT_Close() {
	t := rticker.New(20 * time.Millisecond)

	// consume one tick
	<-t.C

	t.Close()
	fmt.Println("closed:", t.IsClosed())

	// Output:
	// closed: true
}
