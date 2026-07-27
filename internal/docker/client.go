// Package docker provides a client for interacting with the Docker daemon
// via the Unix socket at /var/run/docker.sock.
//
// It implements:
//   - Container stats (cgroups telemetry) via the /containers/{id}/stats API
//   - Log streaming with 8-byte binary header demultiplexing
//   - Docker Compose deployment via os/exec
package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Client talks to the Docker daemon over its Unix socket.
type Client struct {
	http *http.Client
	host string // Unix socket path
}

// NewClient creates a Docker client that dials the given Unix socket.
// Pass "" to use the default /var/run/docker.sock.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = "/var/run/docker.sock"
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
		MaxIdleConns:    5,
		IdleConnTimeout: 30 * time.Second,
	}

	return &Client{
		http: &http.Client{Transport: transport},
		host: socketPath,
	}
}

// apiURL builds a URL for the Docker API. The host is "http://docker"
// because the transport dials Unix directly — the hostname is irrelevant.
func apiURL(path string) string {
	return "http://docker" + path
}

// ---------- Container Stats -------------------------------------------------

// ContainerStats holds a snapshot of container resource usage.
type ContainerStats struct {
	ContainerID string  `json:"container_id"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemUsageMB  float64 `json:"mem_usage_mb"`
	MemLimitMB  float64 `json:"mem_limit_mb"`
	MemPercent  float64 `json:"mem_percent"`
	NetRxBytes  uint64  `json:"net_rx_bytes"`
	NetTxBytes  uint64  `json:"net_tx_bytes"`
	PIDs        uint64  `json:"pids"`
	Timestamp   string  `json:"timestamp"`
}

// statsJSON is a minimal representation of Docker's /containers/{id}/stats response.
type statsJSON struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint64 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

// Stats fetches a single stats snapshot for the given container.
// Uses stream=false to get a one-shot response.
func (c *Client) Stats(ctx context.Context, containerID string) (*ContainerStats, error) {
	url := apiURL(fmt.Sprintf("/containers/%s/stats?stream=false", containerID))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("docker: stats request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker: stats %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker: stats %s: HTTP %d: %s", containerID, resp.StatusCode, body)
	}

	var raw statsJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("docker: stats decode: %w", err)
	}

	return parseStats(containerID, &raw), nil
}

// StatsStream opens a streaming stats connection and sends parsed snapshots
// to the provided channel. It blocks until ctx is cancelled or an error occurs.
func (c *Client) StatsStream(ctx context.Context, containerID string, ch chan<- *ContainerStats) error {
	url := apiURL(fmt.Sprintf("/containers/%s/stats?stream=true", containerID))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("docker: stats stream request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker: stats stream %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker: stats stream %s: HTTP %d: %s", containerID, resp.StatusCode, body)
	}

	dec := json.NewDecoder(resp.Body)
	for {
		var raw statsJSON
		if err := dec.Decode(&raw); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("docker: stats stream decode: %w", err)
		}

		select {
		case ch <- parseStats(containerID, &raw):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func parseStats(containerID string, raw *statsJSON) *ContainerStats {
	s := &ContainerStats{
		ContainerID: containerID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	// CPU percentage: delta usage / delta system * number of CPUs * 100
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)
	cpus := raw.CPUStats.OnlineCPUs
	if cpus == 0 {
		cpus = 1
	}
	if sysDelta > 0 && cpuDelta > 0 {
		s.CPUPercent = (cpuDelta / sysDelta) * float64(cpus) * 100.0
	}

	// Memory
	s.MemUsageMB = float64(raw.MemoryStats.Usage) / 1024 / 1024
	s.MemLimitMB = float64(raw.MemoryStats.Limit) / 1024 / 1024
	if raw.MemoryStats.Limit > 0 {
		s.MemPercent = float64(raw.MemoryStats.Usage) / float64(raw.MemoryStats.Limit) * 100.0
	}

	// Network (aggregate all interfaces)
	for _, n := range raw.Networks {
		s.NetRxBytes += n.RxBytes
		s.NetTxBytes += n.TxBytes
	}

	s.PIDs = raw.PidsStats.Current
	return s
}

// ---------- Log Streaming ---------------------------------------------------

// LogEntry represents a single line of container log output.
type LogEntry struct {
	Stream    string `json:"stream"` // "stdout" or "stderr"
	Timestamp string `json:"timestamp,omitempty"`
	Line      string `json:"line"`
}

// Logs streams container logs, demultiplexing the 8-byte binary headers
// that Docker uses to separate stdout and stderr. Results are sent to ch.
// It blocks until ctx is cancelled or an error occurs.
//
// Docker log header format (8 bytes):
//
//	[0]    stream type: 0=stdin, 1=stdout, 2=stderr
//	[1-3]  padding (zeros)
//	[4-7]  uint32 big-endian frame size
func (c *Client) Logs(ctx context.Context, containerID string, follow bool, tail string, ch chan<- *LogEntry) error {
	url := apiURL(fmt.Sprintf("/containers/%s/logs?stdout=true&stderr=true&follow=%t&tail=%s&timestamps=true",
		containerID, follow, tail))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("docker: logs request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker: logs %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker: logs %s: HTTP %d: %s", containerID, resp.StatusCode, body)
	}

	return demuxLogs(ctx, resp.Body, ch)
}

// demuxLogs reads Docker's multiplexed log stream and parses each frame.
func demuxLogs(ctx context.Context, r io.Reader, ch chan<- *LogEntry) error {
	header := make([]byte, 8)
	for {
		// Read the 8-byte header.
		if _, err := io.ReadFull(r, header); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("docker: log header read: %w", err)
		}

		streamType := header[0]
		frameSize := binary.BigEndian.Uint32(header[4:8])

		// Read the frame payload.
		payload := make([]byte, frameSize)
		if _, err := io.ReadFull(r, payload); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("docker: log payload read: %w", err)
		}

		stream := "stdout"
		if streamType == 2 {
			stream = "stderr"
		}

		// Docker timestamps format: "2006-01-02T15:04:05.999999999Z "
		line := string(payload)
		var ts string
		if len(line) > 31 && line[30] == ' ' {
			ts = line[:30]
			line = line[31:]
		}

		entry := &LogEntry{
			Stream:    stream,
			Timestamp: ts,
			Line:      line,
		}

		select {
		case ch <- entry:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
