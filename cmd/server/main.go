// DevPnl — lightweight deployment panel.
//
// This is the main entry point. It:
//  1. Acquires a listener via systemd socket activation or TCP fallback.
//  2. Wraps routes with an active-request tracker.
//  3. Arms a 5-minute idle timer that gracefully shuts down the server
//     when no requests are in flight (scale-to-zero).
//  4. Serves the embedded Svelte SPA on all non-API routes.
package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VedantJJA/devpnl/internal/caddy"
	"github.com/VedantJJA/devpnl/internal/db"
	"github.com/VedantJJA/devpnl/internal/docker"
	"github.com/VedantJJA/devpnl/internal/server"
	"github.com/VedantJJA/devpnl/internal/sys"
	"github.com/VedantJJA/devpnl/ui"
)

const (
	// defaultAddr is the listen address when not using socket activation.
	defaultAddr = ":8090"

	// idleTimeout is how long the server waits with zero active requests
	// before initiating a graceful shutdown.
	idleTimeout = 5 * time.Minute

	// shutdownGrace is the maximum time allowed for in-flight requests
	// to complete during shutdown.
	shutdownGrace = 30 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// --- Listener -----------------------------------------------------------
	addr := defaultAddr
	if v := os.Getenv("DEVPNL_ADDR"); v != "" {
		addr = v
	}

	ln, err := sys.Listener(addr)
	if err != nil {
		log.Fatalf("listener: %v", err)
	}
	log.Printf("listening on %s", ln.Addr())

	// --- Database -------------------------------------------------------------
	dbPath := "devpnl.db"
	if v := os.Getenv("DEVPNL_DB"); v != "" {
		dbPath = v
	}
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()
	// --- Docker Client -------------------------------------------------------
	dockSocket := "/var/run/docker.sock"
	if v := os.Getenv("DEVPNL_DOCKER_SOCKET"); v != "" {
		dockSocket = v
	}
	dockClient := docker.NewClient(dockSocket)

	// --- Caddy Client --------------------------------------------------------
	caddyAdmin := "http://localhost:2019"
	if v := os.Getenv("DEVPNL_CADDY_ADMIN"); v != "" {
		caddyAdmin = v
	}
	caddyClient := caddy.NewClient(caddyAdmin)
	_ = caddyClient // used when deploying containers to register routes

	// --- Context & Idle Shutdown -------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	idle := server.NewIdleShutdown(idleTimeout, cancel)

	tracker := server.NewTracker(
		idle.ResetIdle,  // onIdle  — restart the countdown
		idle.CancelIdle, // onBusy  — cancel while requests are in flight
	)

	// --- Routes -------------------------------------------------------------
	mux := http.NewServeMux()

	// Health check (useful for Caddy, load balancers, etc.)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API root.
	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"0.1.0","phase":5}`))
	})

	// On-Demand TLS verification endpoint.
	// Caddy calls GET /ask?domain=<fqdn> during TLS handshakes.
	// Return 200 if the domain is in our DB, 404 otherwise.
	mux.HandleFunc("GET /ask", func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		if domain == "" {
			http.Error(w, `{"error":"missing domain param"}`, http.StatusBadRequest)
			return
		}

		exists, err := database.DomainExists(r.Context(), domain)
		if err != nil {
			log.Printf("ask: domain check error: %v", err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}

		if !exists {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"allowed":false}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"allowed":true}`))
	})

	// WebSocket endpoints for real-time container telemetry.
	mux.HandleFunc("/ws/stats", docker.HandleStatsWS(dockClient))
	mux.HandleFunc("/ws/logs", docker.HandleLogsWS(dockClient))

	// --- Embedded SPA -------------------------------------------------------
	uiContent := ui.FS()
	fileServer := http.FileServer(http.FS(uiContent))

	// SPA handler: try serving the file; if it doesn't exist, serve the
	// fallback 200.html so client-side routing can take over.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:] // strip leading slash for fs.Stat
		}

		if _, err := fs.Stat(uiContent, path); err != nil {
			// File not found — serve the SPA fallback.
			r.URL.Path = "/200.html"
		}

		fileServer.ServeHTTP(w, r)
	})

	// Wrap the mux with request tracking middleware.
	handler := tracker.Middleware(mux)

	srv := &http.Server{
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- Signal handling ----------------------------------------------------
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigCh:
			log.Printf("received %s — shutting down", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	// --- Serve --------------------------------------------------------------
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	// Block until context is cancelled (by idle timer or signal).
	<-ctx.Done()
	idle.Stop()

	log.Println("shutting down gracefully…")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
