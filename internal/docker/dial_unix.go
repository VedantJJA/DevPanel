//go:build !windows

package docker

import (
	"context"
	"net"
)

// dialDocker connects to the Docker daemon via a Unix socket on Linux/macOS.
func dialDocker(ctx context.Context, socketPath string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
}
