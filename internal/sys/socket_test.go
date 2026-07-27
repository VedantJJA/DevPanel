package sys

import (
	"net"
	"os"
	"testing"
)

func TestListener_FallbackToTCP(t *testing.T) {
	// Ensure env vars are unset so we hit the fallback path.
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")
	os.Unsetenv("LISTEN_FDNAMES")

	ln, err := Listener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listener() error: %v", err)
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCPAddr, got %T", ln.Addr())
	}
	if addr.Port == 0 {
		t.Fatal("expected a non-zero port from fallback listener")
	}
}
