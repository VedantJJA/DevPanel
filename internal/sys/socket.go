// Package sys handles systemd socket activation and listener setup.
//
// When running under systemd with socket activation, the go-systemd
// activation package retrieves pre-opened listeners from inherited file
// descriptors. When socket activation is not detected (local dev), this
// package falls back to a standard net.Listen on the given address.
package sys

import (
	"fmt"
	"log"
	"net"

	"github.com/coreos/go-systemd/v22/activation"
)

// Listener returns a net.Listener appropriate for the current environment.
//
// Under systemd socket activation it returns the first inherited listener.
// Otherwise it falls back to binding addr via TCP (for local development).
func Listener(addr string) (net.Listener, error) {
	listeners, err := activation.Listeners()
	if err != nil {
		return nil, fmt.Errorf("sys: activation.Listeners: %w", err)
	}

	if len(listeners) > 0 {
		// Use the first socket passed by systemd.
		log.Printf("sys: using systemd socket activation (fd count: %d)", len(listeners))
		return listeners[0], nil
	}

	// No systemd sockets — fall back to standard TCP for local dev.
	log.Printf("sys: no systemd sockets, falling back to tcp/%s", addr)
	return net.Listen("tcp", addr)
}
