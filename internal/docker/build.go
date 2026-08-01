package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// nowRFC3339 returns the current UTC time formatted as RFC 3339.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ---------------------------------------------------------------------------
// buildSink — bridges BlueprintOrchestrator.Sink into the LogManager
// ---------------------------------------------------------------------------

// newBuildSink returns a LogSink that writes every log line into the
// per-project LogBuffer AND into the globalLogBroadcaster (SSE clients).
func newBuildSink(projectID string) LogSink {
	buf := globalLogManager.GetOrCreate(projectID)
	return func(pid, stage, service, message, level string) {
		buf.AppendLine(stage, service, message, level)
		// Also fan-out to SSE clients that are already connected.
		globalLogBroadcaster.Broadcast(pid, stage, service, message, level)
	}
}

// ---------------------------------------------------------------------------
// RunProjectDeploy — called by HTTP handler and trigger handler
// ---------------------------------------------------------------------------

// RunProjectDeploy clears existing logs, then orchestrates a full blueprint
// deploy for the given project in the background. Progress is written both to
// globalLogManager (WebSocket polling) and globalLogBroadcaster (SSE).
func RunProjectDeploy(ctx context.Context, client *Client, projectID string, bp *Blueprint, repoURL string) {
	// Clear previous build logs so the frontend receives a "clear" signal.
	globalLogManager.ClearProject(projectID)

	sink := newBuildSink(projectID)

	orch := NewBlueprintOrchestrator(client)
	orch.ProjectID = projectID
	orch.Sink = sink

	sink(projectID, "init", "engine", "Build pipeline starting…", "info")

	result, err := orch.DeployOrchestrate(ctx, bp, repoURL)
	if err != nil {
		sink(projectID, "deploy", "engine", fmt.Sprintf("Deployment failed: %v", err), "error")
		return
	}

	msg := fmt.Sprintf("Deployment succeeded. Services: %v", result.Services)
	sink(projectID, "deploy", "engine", msg, "success")
}

// ---------------------------------------------------------------------------
// logStreamWriter — io.Writer that parses Docker JSON build stream lines
// ---------------------------------------------------------------------------

// logStreamWriter implements io.Writer; it parses Docker build JSON lines
// and appends them to a LogBuffer, stripping ANSI colour codes.
type logStreamWriter struct {
	buf     *LogBuffer
	service string
}

func newLogStreamWriter(buf *LogBuffer, service string) *logStreamWriter {
	return &logStreamWriter{buf: buf, service: service}
}

func (w *logStreamWriter) Write(p []byte) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(p)))
	for scanner.Scan() {
		raw := scanner.Text()
		// Docker build API streams JSON objects: {"stream":"..."} or {"error":"..."}
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if json.Unmarshal([]byte(raw), &msg) == nil {
			if msg.Stream != "" {
				line := stripANSICodes(strings.TrimRight(msg.Stream, "\r\n"))
				if line != "" {
					w.buf.AppendLine("build", w.service, line, "info")
				}
			}
			if msg.Error != "" {
				w.buf.AppendLine("build", w.service, msg.Error, "error")
			}
		} else if raw != "" {
			w.buf.AppendLine("build", w.service, stripANSICodes(raw), "info")
		}
	}
	return len(p), nil
}

// stripANSICodes removes ANSI terminal escape sequences from a string.
func stripANSICodes(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			// Escape sequences end with a letter
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
