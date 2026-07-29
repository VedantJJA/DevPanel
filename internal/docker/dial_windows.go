//go:build windows

package docker

import (
	"context"
	"net"

	winio "github.com/Microsoft/go-winio"
)

// dialDocker connects to the Docker daemon via a Windows named pipe.
func dialDocker(ctx context.Context, pipePath string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, pipePath)
}
