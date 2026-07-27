package docker

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

// WSMessage is the envelope sent to WebSocket clients.
type WSMessage struct {
	Type string      `json:"type"` // "stats" | "log" | "error"
	Data interface{} `json:"data"`
}

// HandleStatsWS returns an http.HandlerFunc that upgrades to WebSocket
// and streams container stats for the given container ID.
//
// Query params:
//   - id: container ID (required)
//   - interval: stats poll interval in seconds (default: 2)
func HandleStatsWS(dockerClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		containerID := r.URL.Query().Get("id")
		if containerID == "" {
			http.Error(w, `{"error":"missing id param"}`, http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // Allow any origin during dev
		})
		if err != nil {
			log.Printf("ws: accept error: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "done")

		ctx := conn.CloseRead(r.Context())

		// Stream stats using Docker's streaming API.
		statsCh := make(chan *ContainerStats, 8)
		errCh := make(chan error, 1)

		go func() {
			errCh <- dockerClient.StatsStream(ctx, containerID, statsCh)
		}()

		for {
			select {
			case stats, ok := <-statsCh:
				if !ok {
					return
				}
				msg := WSMessage{Type: "stats", Data: stats}
				data, _ := json.Marshal(msg)
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					return
				}

			case err := <-errCh:
				if err != nil && ctx.Err() == nil {
					log.Printf("ws: stats stream error: %v", err)
					msg := WSMessage{Type: "error", Data: err.Error()}
					data, _ := json.Marshal(msg)
					conn.Write(ctx, websocket.MessageText, data)
				}
				return

			case <-ctx.Done():
				return
			}
		}
	}
}

// HandleLogsWS returns an http.HandlerFunc that upgrades to WebSocket
// and streams container logs for the given container ID.
//
// Query params:
//   - id: container ID (required)
//   - tail: number of lines to tail (default: "100")
func HandleLogsWS(dockerClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		containerID := r.URL.Query().Get("id")
		if containerID == "" {
			http.Error(w, `{"error":"missing id param"}`, http.StatusBadRequest)
			return
		}

		tail := r.URL.Query().Get("tail")
		if tail == "" {
			tail = "100"
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ws: accept error: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "done")

		ctx := conn.CloseRead(r.Context())

		logCh := make(chan *LogEntry, 64)
		errCh := make(chan error, 1)

		go func() {
			errCh <- dockerClient.Logs(ctx, containerID, true, tail, logCh)
		}()

		// Batch log entries to avoid flooding the WebSocket.
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		var batch []*LogEntry

		for {
			select {
			case entry, ok := <-logCh:
				if !ok {
					// Flush remaining.
					if len(batch) > 0 {
						flushLogs(ctx, conn, batch)
					}
					return
				}
				batch = append(batch, entry)

			case <-ticker.C:
				if len(batch) > 0 {
					if err := flushLogs(ctx, conn, batch); err != nil {
						return
					}
					batch = batch[:0]
				}

			case err := <-errCh:
				// Flush remaining.
				if len(batch) > 0 {
					flushLogs(ctx, conn, batch)
				}
				if err != nil && ctx.Err() == nil {
					log.Printf("ws: log stream error: %v", err)
					msg := WSMessage{Type: "error", Data: err.Error()}
					data, _ := json.Marshal(msg)
					conn.Write(ctx, websocket.MessageText, data)
				}
				return

			case <-ctx.Done():
				return
			}
		}
	}
}

func flushLogs(ctx context.Context, conn *websocket.Conn, entries []*LogEntry) error {
	msg := WSMessage{Type: "log", Data: entries}
	data, _ := json.Marshal(msg)
	return conn.Write(ctx, websocket.MessageText, data)
}
