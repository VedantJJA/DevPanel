package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// ---------- Tracker tests ----------

func TestTracker_CountsRequests(t *testing.T) {
	tracker := NewTracker(nil, nil)

	if got := tracker.Active(); got != 0 {
		t.Fatalf("expected 0 active, got %d", got)
	}

	// Block a request inside the handler so we can observe the count.
	started := make(chan struct{})
	release := make(chan struct{})

	handler := tracker.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}))

	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)
	}()

	<-started
	if got := tracker.Active(); got != 1 {
		t.Fatalf("expected 1 active, got %d", got)
	}

	close(release)
	// Small sleep to let the deferred Add(-1) complete.
	time.Sleep(10 * time.Millisecond)

	if got := tracker.Active(); got != 0 {
		t.Fatalf("expected 0 active after request, got %d", got)
	}
}

func TestTracker_CallbacksFire(t *testing.T) {
	var mu sync.Mutex
	var idleCalls, busyCalls int

	tracker := NewTracker(
		func() { mu.Lock(); idleCalls++; mu.Unlock() },
		func() { mu.Lock(); busyCalls++; mu.Unlock() },
	)

	handler := tracker.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)

	mu.Lock()
	defer mu.Unlock()

	if busyCalls != 1 {
		t.Errorf("expected 1 onBusy call, got %d", busyCalls)
	}
	if idleCalls != 1 {
		t.Errorf("expected 1 onIdle call, got %d", idleCalls)
	}
}

// ---------- IdleShutdown tests ----------

func TestIdleShutdown_FiresAfterTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = NewIdleShutdown(50*time.Millisecond, cancel)

	select {
	case <-ctx.Done():
		// Success — idle timer fired.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("idle shutdown did not fire within expected window")
	}
}

func TestIdleShutdown_CancelPreventsShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	is := NewIdleShutdown(50*time.Millisecond, cancel)
	is.CancelIdle()

	select {
	case <-ctx.Done():
		t.Fatal("context was cancelled despite CancelIdle")
	case <-time.After(150 * time.Millisecond):
		// Success — timer did not fire.
	}
}

func TestIdleShutdown_ResetExtendsTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	is := NewIdleShutdown(80*time.Millisecond, cancel)

	// Reset at 40ms — should push the deadline to 120ms total.
	time.Sleep(40 * time.Millisecond)
	is.ResetIdle()

	// At 90ms total the original timer would have fired, but shouldn't.
	time.Sleep(50 * time.Millisecond)
	select {
	case <-ctx.Done():
		t.Fatal("context cancelled too early after reset")
	default:
	}

	// By 160ms the reset timer should have fired.
	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("idle shutdown did not fire after reset")
	}
}
