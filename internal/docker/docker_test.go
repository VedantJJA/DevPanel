package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
)

// buildLogFrame constructs a Docker log frame with the 8-byte header.
func buildLogFrame(streamType byte, payload string) []byte {
	var buf bytes.Buffer
	// Header: [streamType, 0, 0, 0, size(4 bytes big-endian)]
	header := make([]byte, 8)
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	buf.Write(header)
	buf.WriteString(payload)
	return buf.Bytes()
}

func TestDemuxLogs_StdoutAndStderr(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildLogFrame(1, "2026-01-01T00:00:00.000000000Z hello from stdout\n"))
	buf.Write(buildLogFrame(2, "2026-01-01T00:00:01.000000000Z error from stderr\n"))
	buf.Write(buildLogFrame(1, "short line\n"))

	ch := make(chan *LogEntry, 10)
	ctx := context.Background()

	err := demuxLogs(ctx, &buf, ch)
	if err != nil {
		t.Fatalf("demuxLogs: %v", err)
	}
	close(ch)

	var entries []*LogEntry
	for e := range ch {
		entries = append(entries, e)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// First entry: stdout with timestamp
	if entries[0].Stream != "stdout" {
		t.Errorf("entry 0: expected stream=stdout, got %q", entries[0].Stream)
	}
	if entries[0].Timestamp != "2026-01-01T00:00:00.000000000Z" {
		t.Errorf("entry 0: unexpected timestamp %q", entries[0].Timestamp)
	}
	if entries[0].Line != "hello from stdout\n" {
		t.Errorf("entry 0: unexpected line %q", entries[0].Line)
	}

	// Second entry: stderr
	if entries[1].Stream != "stderr" {
		t.Errorf("entry 1: expected stream=stderr, got %q", entries[1].Stream)
	}

	// Third entry: short line (no Docker timestamp)
	if entries[2].Stream != "stdout" {
		t.Errorf("entry 2: expected stream=stdout, got %q", entries[2].Stream)
	}
}

func TestDemuxLogs_EmptyInput(t *testing.T) {
	ch := make(chan *LogEntry, 10)
	err := demuxLogs(context.Background(), &bytes.Buffer{}, ch)
	if err != nil {
		t.Fatalf("demuxLogs on empty: %v", err)
	}
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 entries from empty input, got %d", count)
	}
}

func TestDemuxLogs_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	var buf bytes.Buffer
	buf.Write(buildLogFrame(1, "2026-01-01T00:00:00.000000000Z will be ignored\n"))

	ch := make(chan *LogEntry, 10)
	err := demuxLogs(ctx, &buf, ch)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestParseStats(t *testing.T) {
	raw := &statsJSON{}
	raw.CPUStats.CPUUsage.TotalUsage = 200
	raw.CPUStats.SystemCPUUsage = 1000
	raw.CPUStats.OnlineCPUs = 2
	raw.PreCPUStats.CPUUsage.TotalUsage = 100
	raw.PreCPUStats.SystemCPUUsage = 500

	raw.MemoryStats.Usage = 100 * 1024 * 1024 // 100 MB
	raw.MemoryStats.Limit = 512 * 1024 * 1024 // 512 MB

	raw.PidsStats.Current = 42

	s := parseStats("test123", raw)

	if s.ContainerID != "test123" {
		t.Errorf("expected ContainerID=test123, got %q", s.ContainerID)
	}

	// CPU: (200-100) / (1000-500) * 2 * 100 = 40%
	if s.CPUPercent < 39.9 || s.CPUPercent > 40.1 {
		t.Errorf("expected CPUPercent≈40, got %.2f", s.CPUPercent)
	}

	if s.MemUsageMB < 99.9 || s.MemUsageMB > 100.1 {
		t.Errorf("expected MemUsageMB≈100, got %.2f", s.MemUsageMB)
	}

	if s.PIDs != 42 {
		t.Errorf("expected PIDs=42, got %d", s.PIDs)
	}
}
