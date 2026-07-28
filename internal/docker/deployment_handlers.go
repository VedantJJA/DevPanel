package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/VedantJJA/devpnl/internal/db"
)

// TriggerDeploymentRequest represents the POST payload to initiate deployment.
type TriggerDeploymentRequest struct {
	Project   string     `json:"project"`
	RepoURL   string     `json:"repo_url"`
	Blueprint *Blueprint `json:"blueprint"`
}

// TriggerDeploymentResponse represents the instant 202 Accepted response.
type TriggerDeploymentResponse struct {
	ProjectID string `json:"project_id"`
	Message   string `json:"message"`
	Status    string `json:"status"`
}

// LogEvent represents a single SSE log message payload.
type LogEvent struct {
	Timestamp string `json:"timestamp"`
	Stage     string `json:"stage"` // "clone", "build", "deploy", "runtime", "system"
	Service   string `json:"service"`
	Message   string `json:"message"`
	Level     string `json:"level"` // "info", "warn", "error", "success"
}

// LogBroadcaster manages active SSE subscriber channels per project ID.
type LogBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[string][]chan LogEvent
	history     map[string][]LogEvent
}

var globalLogBroadcaster = &LogBroadcaster{
	subscribers: make(map[string][]chan LogEvent),
	history:     make(map[string][]LogEvent),
}

// Broadcast sends a log event to all connected SSE clients for a project.
func (b *LogBroadcaster) Broadcast(projectID string, stage, service, message, level string) {
	evt := LogEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Stage:     stage,
		Service:   service,
		Message:   message,
		Level:     level,
	}

	b.mu.Lock()
	b.history[projectID] = append(b.history[projectID], evt)
	subs := append([]chan LogEvent(nil), b.subscribers[projectID]...)
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
			// Non-blocking send if subscriber channel buffer is full
		}
	}
}

// Subscribe attaches a new SSE client channel for a project ID.
func (b *LogBroadcaster) Subscribe(projectID string) (chan LogEvent, func()) {
	ch := make(chan LogEvent, 100)

	b.mu.Lock()
	b.subscribers[projectID] = append(b.subscribers[projectID], ch)
	pastLogs := append([]LogEvent(nil), b.history[projectID]...)
	b.mu.Unlock()

	// Immediately send past log history to catching up client
	go func() {
		for _, evt := range pastLogs {
			ch <- evt
		}
	}()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subscribers[projectID]
		for i, sub := range subs {
			if sub == ch {
				b.subscribers[projectID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, unsubscribe
}

// HandleTriggerDeployment catches POST /api/deployments/trigger
func HandleTriggerDeployment(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
			return
		}

		var req TriggerDeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON request body"})
			return
		}

		if req.Project == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Project name is required"})
			return
		}

		projectID := sanitizeName(req.Project)

		// Save project record to SQLite DB if database available
		if database != nil {
			rec := &db.BlueprintRecord{
				ID:           projectID,
				Name:         req.Project,
				RepoURL:      req.RepoURL,
				Status:       "deploying",
				ServiceCount: len(req.Blueprint.Services),
			}
			_ = database.CreateBlueprint(r.Context(), rec)
		}

		// Asynchronous Goroutine for Docker build and deployment engine
		go func() {
			bgCtx := context.Background()
			globalLogBroadcaster.Broadcast(projectID, "system", "engine", fmt.Sprintf("Initiating deployment pipeline for project %q...", req.Project), "info")

			orchestrator := NewBlueprintOrchestrator(dockClient)

			globalLogBroadcaster.Broadcast(projectID, "clone", "git", fmt.Sprintf("Cloning repository %s...", req.RepoURL), "info")
			time.Sleep(500 * time.Millisecond)

			if req.Blueprint == nil || len(req.Blueprint.Services) == 0 {
				globalLogBroadcaster.Broadcast(projectID, "system", "engine", "No services defined in blueprint", "error")
				return
			}

			for sName, sCfg := range req.Blueprint.Services {
				globalLogBroadcaster.Broadcast(projectID, "build", sName, fmt.Sprintf("Building service container '%s' (type: %s)...", sName, sCfg.Type), "info")
				time.Sleep(300 * time.Millisecond)
				globalLogBroadcaster.Broadcast(projectID, "build", sName, fmt.Sprintf("Compiling image for %s...", sName), "info")
			}

			res, err := orchestrator.DeployOrchestrate(bgCtx, req.Blueprint, req.RepoURL)
			if err != nil {
				log.Printf("deployment-engine: project %s failed: %v", projectID, err)
				globalLogBroadcaster.Broadcast(projectID, "deploy", "engine", fmt.Sprintf("Deployment failed: %v", err), "error")
				if database != nil {
					_ = database.CreateBlueprint(bgCtx, &db.BlueprintRecord{
						ID:      projectID,
						Name:    req.Project,
						RepoURL: req.RepoURL,
						Status:  "error",
					})
				}
				return
			}

			globalLogBroadcaster.Broadcast(projectID, "deploy", "engine", fmt.Sprintf("Successfully provisioned %d services! Network active.", len(res.Services)), "success")
			globalLogBroadcaster.Broadcast(projectID, "runtime", "engine", fmt.Sprintf("Container stack for project '%s' is live and online.", req.Project), "success")

			if database != nil {
				_ = database.CreateBlueprint(bgCtx, &db.BlueprintRecord{
					ID:           projectID,
					Name:         req.Project,
					RepoURL:      req.RepoURL,
					Status:       "active",
					ServiceCount: len(res.Services),
				})
			}
		}()

		// Instant HTTP 202 Accepted Response
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(TriggerDeploymentResponse{
			ProjectID: projectID,
			Message:   "Deployment process started asynchronously",
			Status:    "deploying",
		})
	}
}

// HandleDeploymentLogsSSE streams real-time Server-Sent Events (SSE) to frontend.
func HandleDeploymentLogsSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set headers for Server-Sent Events (SSE)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Extract project ID from path or query parameter
		projectID := strings.TrimPrefix(r.URL.Path, "/api/deployments/")
		projectID = strings.TrimSuffix(projectID, "/logs/sse")
		if qID := r.URL.Query().Get("id"); qID != "" {
			projectID = qID
		}

		if projectID == "" {
			http.Error(w, "Project ID is required", http.StatusBadRequest)
			return
		}

		ch, unsubscribe := globalLogBroadcaster.Subscribe(projectID)
		defer unsubscribe()

		// Initial connection event
		fmt.Fprintf(w, "event: connected\ndata: %s\n\n", projectID)
		flusher.Flush()

		ctx := r.Context()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// SSE ping heartbeat to prevent proxy timeout
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case evt, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(evt)
				if err == nil {
					fmt.Fprintf(w, "data: %s\n\n", string(data))
					flusher.Flush()
				}
			}
		}
	}
}
