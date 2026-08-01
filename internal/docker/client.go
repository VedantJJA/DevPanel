// Package docker provides a client for interacting with the Docker daemon
// via the Unix socket at /var/run/docker.sock.
//
// It implements:
//   - Container stats (cgroups telemetry) via the /containers/{id}/stats API
//   - Container management (list, start, stop, info) via Docker Engine API
//   - Log streaming with 8-byte binary header demultiplexing
//   - Docker Compose deployment via os/exec
package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Client talks to the Docker daemon over its Unix socket or named pipe.
type Client struct {
	http *http.Client
	host string // socket path or named pipe path
}

// NewClient creates a Docker client that dials the Docker daemon.
// On Windows it uses the named pipe; on Linux/macOS it uses the Unix socket.
// Pass "" to use the OS-appropriate default.
func NewClient(socketPath string) *Client {
	if socketPath == "" || socketPath == "/var/run/docker.sock" {
		if runtime.GOOS == "windows" {
			socketPath = `//./pipe/docker_engine`
		} else {
			socketPath = "/var/run/docker.sock"
		}
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialDocker(ctx, socketPath)
		},
		MaxIdleConns:    5,
		IdleConnTimeout: 30 * time.Second,
	}

	log.Printf("docker: client initialized (host=%s, os=%s)", socketPath, runtime.GOOS)

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

// ---------- Docker Engine API Types -----------------------------------------

// ContainerSummary represents a container returned by /containers/json.
type ContainerSummary struct {
	ID      string `json:"Id"`
	Names   []string `json:"Names"`
	Image   string `json:"Image"`
	State   string `json:"State"`
	Status  string            `json:"Status"`
	Ports   []Port            `json:"Ports"`
	Labels  map[string]string `json:"Labels"`
	Created int64  `json:"Created"`
}

// Port represents a mapped port on a container.
type Port struct {
	IP          string `json:"IP"`
	PrivatePort uint16 `json:"PrivatePort"`
	PublicPort  uint16 `json:"PublicPort"`
	Type        string `json:"Type"`
}

// VolumeSummary represents a volume returned by /volumes.
type VolumeSummary struct {
	Name       string `json:"Name"`
	Driver     string `json:"Driver"`
	Mountpoint string `json:"Mountpoint"`
	CreatedAt  string `json:"CreatedAt"`
	Scope      string `json:"Scope"`
}

type volumeListResponse struct {
	Volumes []VolumeSummary `json:"Volumes"`
}

// SystemInfo represents Docker host information returned by /info.
type SystemInfo struct {
	Containers        int    `json:"Containers"`
	ContainersRunning int    `json:"ContainersRunning"`
	ContainersStopped int    `json:"ContainersStopped"`
	NCPU              int    `json:"NCPU"`
	MemTotal          int64  `json:"MemTotal"`
	OperatingSystem   string `json:"OperatingSystem"`
	Architecture      string `json:"Architecture"`
}

// ---------- System & Container Management API ------------------------------

// SystemInfo fetches live system and container counts from Docker daemon.
func (c *Client) SystemInfo(ctx context.Context) (*SystemInfo, error) {
	url := apiURL("/info")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("docker: info request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker: info call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker: info HTTP %d: %s", resp.StatusCode, body)
	}

	var info SystemInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("docker: info decode: %w", err)
	}

	return &info, nil
}

// ListContainers retrieves all containers from the Docker daemon.
func (c *Client) ListContainers(ctx context.Context) ([]ContainerSummary, error) {
	url := apiURL("/containers/json?all=true")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("docker: list containers request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker: list containers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker: list containers HTTP %d: %s", resp.StatusCode, body)
	}

	var containers []ContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("docker: list containers decode: %w", err)
	}

	return containers, nil
}

// ListVolumes retrieves all volumes from the Docker daemon.
func (c *Client) ListVolumes(ctx context.Context) ([]VolumeSummary, error) {
	url := apiURL("/volumes")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("docker: list volumes request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker: list volumes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker: list volumes HTTP %d: %s", resp.StatusCode, body)
	}

	var res volumeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("docker: list volumes decode: %w", err)
	}

	return res.Volumes, nil
}

// StartContainer sends a start command for the specified container.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	url := apiURL(fmt.Sprintf("/containers/%s/start", containerID))
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("docker: start container request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker: start container %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker: start container %s HTTP %d: %s", containerID, resp.StatusCode, body)
	}

	return nil
}

// StopContainer sends a stop command for the specified container.
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	url := apiURL(fmt.Sprintf("/containers/%s/stop", containerID))
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("docker: stop container request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker: stop container %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker: stop container %s HTTP %d: %s", containerID, resp.StatusCode, body)
	}

	return nil
}

// RemoveContainer deletes a container. If force is true, sends SIGKILL if running.
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	url := apiURL(fmt.Sprintf("/containers/%s?v=true&force=%t", containerID, force))
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("docker: remove container request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker: remove container %s: %w", containerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		var apiErr struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("%s", apiErr.Message)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	return nil
}

// ContainerCreateConfig represents the request JSON payload for creating a container via Docker API.
type ContainerCreateConfig struct {
	Image        string              `json:"Image"`
	Env          []string            `json:"Env,omitempty"`
	Cmd          []string            `json:"Cmd,omitempty"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts,omitempty"`
	Labels       map[string]string   `json:"Labels,omitempty"`
	HostConfig   HostConfig          `json:"HostConfig,omitempty"`
}

type HostConfig struct {
	PortBindings map[string][]PortBinding `json:"PortBindings,omitempty"`
	Binds        []string                 `json:"Binds,omitempty"`
	AutoRemove   bool                     `json:"AutoRemove,omitempty"`
	NetworkMode  string                   `json:"NetworkMode,omitempty"`
}

type PortBinding struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

// EnsureNetwork ensures that a Docker bridge network exists for inter-container communication.
func (c *Client) EnsureNetwork(ctx context.Context, networkName string) error {
	url := apiURL("/networks/create")
	payload := map[string]interface{}{
		"Name":           networkName,
		"CheckDuplicate": true,
		"Driver":         "bridge",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type ContainerCreateResponse struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

// CreateContainer sends POST /containers/create?name=xxx to Docker daemon.
func (c *Client) CreateContainer(ctx context.Context, name string, config ContainerCreateConfig) (string, error) {
	url := apiURL(fmt.Sprintf("/containers/create?name=%s", name))
	bodyBytes, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal container config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("docker create container %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("docker create container %s HTTP %d: %s", name, resp.StatusCode, b)
	}

	var res ContainerCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}

	return res.ID, nil
}

// RemoveVolume deletes a volume by name. If force is true, removes even if in use.
func (c *Client) RemoveVolume(ctx context.Context, volumeName string, force bool) error {
	url := apiURL(fmt.Sprintf("/volumes/%s?force=%t", volumeName, force))
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("docker: remove volume request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker: remove volume %s: %w", volumeName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		var apiErr struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("%s", apiErr.Message)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	return nil
}

// PruneSystem prunes stopped containers and unused volumes.
func (c *Client) PruneSystem(ctx context.Context) error {
	cUrl := apiURL("/containers/prune")
	cReq, _ := http.NewRequestWithContext(ctx, "POST", cUrl, nil)
	if cResp, err := c.http.Do(cReq); err == nil {
		cResp.Body.Close()
	}

	vUrl := apiURL("/volumes/prune")
	vReq, _ := http.NewRequestWithContext(ctx, "POST", vUrl, nil)
	if vResp, err := c.http.Do(vReq); err == nil {
		vResp.Body.Close()
	}

	return nil
}

// FormatPorts turns a list of Port structs into a human-readable port mapping string.
func FormatPorts(ports []Port) string {
	if len(ports) == 0 {
		return ""
	}
	var pairs []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			pairs = append(pairs, fmt.Sprintf("%d:%d", p.PublicPort, p.PrivatePort))
		} else {
			pairs = append(pairs, fmt.Sprintf("%d", p.PrivatePort))
		}
	}
	return strings.Join(pairs, ", ")
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
func (c *Client) Logs(ctx context.Context, containerID string, follow bool, tail string, since int64, ch chan<- *LogEntry) error {
	url := apiURL(fmt.Sprintf("/containers/%s/logs?stdout=true&stderr=true&follow=%t&tail=%s&timestamps=true",
		containerID, follow, tail))
	
	if since > 0 {
		url += fmt.Sprintf("&since=%d", since)
	}

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
