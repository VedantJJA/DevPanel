package sys

import (
	"net"
	"os"
	"strconv"
	"testing"
)

func TestListener_FallbackToTCP(t *testing.T) {
	// Ensure env vars are unset so we hit the fallback path.
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	ln, err := Listener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listener() error: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	if addr.Port == 0 {
		t.Fatal("expected a non-zero port from fallback listener")
	}
}

func TestFromSystemd_InvalidPID(t *testing.T) {
	t.Setenv("LISTEN_PID", "not-a-number")
	t.Setenv("LISTEN_FDS", "1")

	_, err := fromSystemd()
	if err == nil {
		t.Fatal("expected error for invalid LISTEN_PID")
	}
}

func TestFromSystemd_WrongPID(t *testing.T) {
	// Use a PID that definitely isn't ours.
	t.Setenv("LISTEN_PID", strconv.Itoa(os.Getpid()+999))
	t.Setenv("LISTEN_FDS", "1")

	ln, err := fromSystemd()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ln != nil {
		ln.Close()
		t.Fatal("expected nil listener when PID doesn't match")
	}
}

func TestFromSystemd_NoEnvVars(t *testing.T) {
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	ln, err := fromSystemd()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ln != nil {
		ln.Close()
		t.Fatal("expected nil listener when env vars absent")
	}
}

func TestFromSystemd_ZeroFDs(t *testing.T) {
	t.Setenv("LISTEN_PID", strconv.Itoa(os.Getpid()))
	t.Setenv("LISTEN_FDS", "0")

	_, err := fromSystemd()
	if err == nil {
		t.Fatal("expected error for LISTEN_FDS=0")
	}
}
