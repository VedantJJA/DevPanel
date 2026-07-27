// Package server provides HTTP middleware and scale-to-zero lifecycle management.
package server

import (
	"net/http"
	"sync/atomic"
)

// Tracker keeps a count of in-flight HTTP requests and notifies a callback
// whenever the count transitions to zero.
type Tracker struct {
	active  atomic.Int64
	onIdle  func() // called when active drops to 0
	onBusy  func() // called when active rises from 0 to 1
}

// NewTracker creates a request tracker. onIdle is called each time the
// active count drops to zero. onBusy is called when it rises from zero.
// Either callback may be nil.
func NewTracker(onIdle, onBusy func()) *Tracker {
	return &Tracker{onIdle: onIdle, onBusy: onBusy}
}

// Active returns the current number of in-flight requests.
func (t *Tracker) Active() int64 {
	return t.active.Load()
}

// Middleware returns an http.Handler that wraps next with request tracking.
func (t *Tracker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prev := t.active.Add(1)
		if prev == 1 && t.onBusy != nil {
			t.onBusy()
		}

		defer func() {
			cur := t.active.Add(-1)
			if cur == 0 && t.onIdle != nil {
				t.onIdle()
			}
		}()

		next.ServeHTTP(w, r)
	})
}
