// Package sys handles systemd socket activation and listener setup.
//
// When running under systemd with socket activation, the LISTEN_FDS and
// LISTEN_PID environment variables are set. This package parses them and
// returns file-descriptor-based listeners. When those variables are absent
// (local dev), it falls back to a standard net.Listen on the given address.
package sys

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

const (
	// listenFDsStart is the first file descriptor passed by systemd.
	// Systemd guarantees FDs start at 3 (after stdin, stdout, stderr).
	listenFDsStart = 3
)

// Listener returns a net.Listener appropriate for the current environment.
//
// If LISTEN_FDS and LISTEN_PID are set (systemd socket activation), it
// inherits the first passed file descriptor. Otherwise it binds to addr
// using standard TCP.
func Listener(addr string) (net.Listener, error) {
	if ln, err := fromSystemd(); ln != nil || err != nil {
		return ln, err
	}
	// Fallback: standard listen for local development.
	return net.Listen("tcp", addr)
}

// fromSystemd attempts to retrieve a listener from systemd-passed file
// descriptors. Returns (nil, nil) when not running under socket activation.
func fromSystemd() (net.Listener, error) {
	pidStr := os.Getenv("LISTEN_PID")
	fdsStr := os.Getenv("LISTEN_FDS")

	if pidStr == "" || fdsStr == "" {
		return nil, nil
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil, fmt.Errorf("sys: invalid LISTEN_PID %q: %w", pidStr, err)
	}

	// The PID must match our own process; otherwise, the FDs are not for us.
	if pid != os.Getpid() {
		return nil, nil
	}

	nfds, err := strconv.Atoi(fdsStr)
	if err != nil {
		return nil, fmt.Errorf("sys: invalid LISTEN_FDS %q: %w", fdsStr, err)
	}

	if nfds < 1 {
		return nil, fmt.Errorf("sys: LISTEN_FDS=%d, expected at least 1", nfds)
	}

	// We only use the first passed FD.
	fd := os.NewFile(uintptr(listenFDsStart), "systemd-socket")
	if fd == nil {
		return nil, fmt.Errorf("sys: unable to open fd %d", listenFDsStart)
	}

	ln, err := net.FileListener(fd)
	if err != nil {
		return nil, fmt.Errorf("sys: FileListener on fd %d: %w", listenFDsStart, err)
	}

	// Close the os.File; the listener now owns the underlying descriptor.
	fd.Close()

	// Clear the env vars so child processes don't inherit them.
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	return ln, nil
}
