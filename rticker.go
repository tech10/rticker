// Package rticker provides a resettable Ticker that can be reset to a new duration, and closed when not in use,
// which will subsequently close its receiving channel, helping to prevent goroutines from leaking.
package rticker

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrClosed = errors.New("ticker already closed")

// T is a ticker that wraps time.Timer and allows resetting the interval.
type T struct {
	C         <-chan time.Time   // read-only ticker channel
	cInternal chan time.Time     // internal channel to send ticks on
	wg        sync.WaitGroup     // internal WaitGroup for ticker loop
	interval  time.Duration      // ticker duration
	timer     *time.Timer        // internal timer
	resetCh   chan time.Duration // channel to request a reset
	ctx       context.Context    // context for cancellation
	ctxCancel context.CancelFunc // executed on Close to cancel ctx
	once      sync.Once          // for closing the ticker only one time
}

// NewWithContext creates a resettable t (ticker) with a context.
// ctx must not be nil, or a runtime panic is produced.
// d must not be less than or equal to 0, or a runtime panic is produced.
func NewWithContext(ctx context.Context, d time.Duration) *T {
	if ctx == nil {
		panic("rticker: nil context passed to NewWithContext")
	}
	if d <= 0 {
		panic("rticker: negative or 0 duration")
	}
	timeChan := make(chan time.Time)
	rt := &T{
		C:         timeChan,
		cInternal: timeChan,
		interval:  d,
		resetCh:   make(chan time.Duration),
	}
	rt.ctx, rt.ctxCancel = context.WithCancel(ctx)
	rt.start()
	return rt
}

// New creates a new T (ticker).
// d must not be less than or equal to 0, or a runtime panic is produced.
func New(d time.Duration) *T {
	return NewWithContext(context.Background(), d)
}

// Reset resets the internal timer with the given interval.
// Returns ErrClosed if the ticker is closed.
func (rt *T) Reset(d time.Duration) error {
	select {
	case <-rt.ctx.Done():
		return ErrClosed
	case rt.resetCh <- d:
		return nil
	}
}

// Stop stops the ticker temporarily. It can be reset again, and is merely paused.
// Returns ErrClosed if the ticker is closed.
func (rt *T) Stop() error {
	return rt.Reset(0)
}

// Close stops the ticker permanently and closes the output channel.
// Returns ErrClosed if the ticker is already closed.
func (rt *T) Close() error {
	err := ErrClosed
	rt.once.Do(func() {
		err = nil
		rt.ctxCancel()
		rt.Wait()
		close(rt.cInternal)
		rt.emptyTimer()
	})
	return err
}

// IsClosed checks if the ticker is closed.
func (rt *T) IsClosed() bool {
	select {
	case <-rt.ctx.Done():
		return true
	default:
		return false
	}
}

// Wait waits until the ticker is closed.
func (rt *T) Wait() {
	rt.wg.Wait()
}

// start starts the ticker. For internal use only.
func (rt *T) start() {
	rt.wg.Add(1)
	rt.timer = time.NewTimer(rt.interval)
	go rt.handler()
}

// handler handles the ticker internals.
func (rt *T) handler() {
	defer rt.Close() // needed in case ctx is canceled
	defer rt.wg.Done()

	for {
		select {
		case <-rt.ctx.Done():
			return
		case d := <-rt.resetCh:
			rt.emptyTimer()
			if d > 0 {
				rt.interval = d
				rt.timer.Reset(d)
			}
		case t := <-rt.timer.C:
			if !rt.sendAndRestart(t) {
				return
			}
		}
	}
}

// emptyTimer stops and empties the internal timer.
func (rt *T) emptyTimer() {
	if !rt.timer.Stop() {
		select {
		case <-rt.timer.C:
		default:
		}
	}
}

// restartTimer restarts the internal timer.
// Returns true if restarted, false if not.
func (rt *T) sendAndRestart(t time.Time) bool {
	select {
	case <-rt.ctx.Done():
		return false // context canceled
	case rt.cInternal <- t:
		rt.timer.Reset(rt.interval)
		return true // timer reset
	}
}
