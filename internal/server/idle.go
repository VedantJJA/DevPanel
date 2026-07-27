package server

import (
	"context"
	"log"
	"sync"
	"time"
)

// IdleShutdown monitors request activity and triggers a graceful shutdown
// when the server has been idle (zero active requests) for the configured
// duration.
type IdleShutdown struct {
	timeout  time.Duration
	cancel   context.CancelFunc // cancels the parent server context
	mu       sync.Mutex
	timer    *time.Timer
	stopped  bool
}

// NewIdleShutdown creates an idle monitor. When the idle duration elapses
// without any new requests, cancel is called to initiate shutdown.
func NewIdleShutdown(timeout time.Duration, cancel context.CancelFunc) *IdleShutdown {
	is := &IdleShutdown{
		timeout: timeout,
		cancel:  cancel,
	}
	// Start the initial idle timer — if nothing connects within the
	// timeout after boot, we shut down immediately.
	is.ResetIdle()
	return is
}

// ResetIdle restarts the idle countdown. Called by the Tracker's onIdle
// callback every time active requests drop to zero.
func (is *IdleShutdown) ResetIdle() {
	is.mu.Lock()
	defer is.mu.Unlock()

	if is.stopped {
		return
	}

	if is.timer != nil {
		is.timer.Stop()
	}

	is.timer = time.AfterFunc(is.timeout, func() {
		log.Printf("server idle for %s — initiating graceful shutdown", is.timeout)
		is.cancel()
	})
}

// CancelIdle stops the idle timer. Called by the Tracker's onBusy callback
// when a request arrives while the timer is ticking.
func (is *IdleShutdown) CancelIdle() {
	is.mu.Lock()
	defer is.mu.Unlock()

	if is.timer != nil {
		is.timer.Stop()
		is.timer = nil
	}
}

// Stop permanently disables the idle monitor (used during shutdown).
func (is *IdleShutdown) Stop() {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.stopped = true
	if is.timer != nil {
		is.timer.Stop()
		is.timer = nil
	}
}
