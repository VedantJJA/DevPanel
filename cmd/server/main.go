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
	"path"
	"strings"
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

	// --- 1. Listener Initialization -----------------------------------------
	addr := defaultAddr
	if v := os.Getenv("DEVPNL_ADDR"); v != "" {
		addr = v
	}

	ln, err := sys.Listener(addr)
	if err != nil {
		log.Fatalf("main: listener acquisition failed: %v", err)
	}
	defer ln.Close()
	log.Printf("main: server listening on %s", ln.Addr())

	// --- Database Initialization ---------------------------------------------
	dbPath := "devpnl.db"
	if v := os.Getenv("DEVPNL_DB"); v != "" {
		dbPath = v
	}
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("main: database initialization failed: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("main: error closing database: %v", err)
		}
	}()

	// --- External Clients ----------------------------------------------------
	dockSocket := "/var/run/docker.sock"
	if v := os.Getenv("DEVPNL_DOCKER_SOCKET"); v != "" {
		dockSocket = v
	}
	dockClient := docker.NewClient(dockSocket)

	caddyAdmin := "http://localhost:2019"
	if v := os.Getenv("DEVPNL_CADDY_ADMIN"); v != "" {
		caddyAdmin = v
	}
	caddyClient := caddy.NewClient(caddyAdmin)
	_ = caddyClient // retained for dynamic route management during container lifecycle

	// --- Context & Graceful Lifecycle ---------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Idle monitor triggers context cancellation when 0 active requests persist for idleTimeout.
	idle := server.NewIdleShutdown(idleTimeout, cancel)

	tracker := server.NewTracker(
		idle.ResetIdle,  // onIdle: restart idle countdown
		idle.CancelIdle, // onBusy: cancel idle countdown while requests in-flight
	)

	// --- HTTP Route Registration --------------------------------------------
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API root endpoint
	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"0.1.0","phase":5}`))
	})

	// On-Demand TLS verification endpoint for Caddy
	mux.HandleFunc("GET /ask", func(w http.ResponseWriter, r *http.Request) {
		domainName := r.URL.Query().Get("domain")
		if domainName == "" {
			http.Error(w, `{"error":"missing domain param"}`, http.StatusBadRequest)
			return
		}

		exists, err := database.DomainExists(r.Context(), domainName)
		if err != nil {
			log.Printf("ask: domain check error for %s: %v", domainName, err)
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

	// Dynamic Container, Volume & System Telemetry API
	mux.HandleFunc("/api/containers", docker.HandleListContainers(dockClient, database))
	mux.HandleFunc("/api/volumes", docker.HandleListVolumes(dockClient))
	mux.HandleFunc("/api/system/stats", docker.HandleSystemStats(dockClient))
	// --- Auth Endpoints (Unprotected) ---
	mux.HandleFunc("GET /api/auth/status", docker.HandleAuthStatus(database))
	mux.HandleFunc("POST /api/auth/setup", docker.HandleAuthSetup(database))
	mux.HandleFunc("POST /api/auth/login", docker.HandleAuthLogin(database))
	mux.HandleFunc("POST /api/auth/logout", docker.HandleAuthLogout())

	// --- Protected API Endpoints ---
	apiMux := http.NewServeMux()
	
	// Legacy endpoints
	apiMux.HandleFunc("/api/containers", docker.HandleListContainers(dockClient, database))
	apiMux.HandleFunc("/api/containers/start-all", docker.HandleStartAllContainers(dockClient))
	apiMux.HandleFunc("/api/containers/stop-all", docker.HandleStopAllContainers(dockClient))
	apiMux.HandleFunc("/api/containers/start", docker.HandleStartContainer(dockClient))
	apiMux.HandleFunc("/api/containers/stop", docker.HandleStopContainer(dockClient))
	apiMux.HandleFunc("/api/containers/delete", docker.HandleDeleteContainer(dockClient, database))
	apiMux.HandleFunc("/api/volumes/delete", docker.HandleDeleteVolume(dockClient))
	apiMux.HandleFunc("/api/blueprints", docker.HandleListBlueprints(database))
	apiMux.HandleFunc("/api/blueprints/delete", docker.HandleDeleteBlueprint(database))
	apiMux.HandleFunc("/api/blueprints/deploy", docker.HandleDeployBlueprint(dockClient))
	apiMux.HandleFunc("/api/blueprints/validate", docker.HandleValidateBlueprint(database))
	apiMux.HandleFunc("/api/deployments/trigger", docker.HandleTriggerDeployment(database, dockClient))
	apiMux.HandleFunc("/api/deployments/", docker.HandleDeploymentLogsSSE())

	// New Render-style Project & Scan Endpoints
	apiMux.HandleFunc("GET /api/repos/user", docker.HandleListUserRepos(database))
	apiMux.HandleFunc("POST /api/repos/scan", docker.HandleScanRepo(database))
	apiMux.HandleFunc("POST /api/projects", docker.HandleCreateProject(database))
	apiMux.HandleFunc("GET /api/projects", docker.HandleListProjects(database))
	apiMux.HandleFunc("GET /api/projects/{id}", docker.HandleGetProject(database))
	apiMux.HandleFunc("PATCH /api/projects/{id}/services/{name}", docker.HandleUpdateService(database))
	apiMux.HandleFunc("POST /api/projects/{id}/deploy", docker.HandleTriggerProjectDeploy(database, dockClient))
	apiMux.HandleFunc("GET /api/projects/{id}/deployments", docker.HandleListDeployments(database))
	apiMux.HandleFunc("GET /api/projects/{id}/logs", docker.HandleProjectLogsSSE())
	apiMux.HandleFunc("POST /api/projects/{id}/services/{name}/restart", docker.HandleRestartService(database, dockClient))
	apiMux.HandleFunc("GET /api/projects/{id}/services/{name}/logs", docker.HandleServiceLogsSSE(dockClient))
	apiMux.HandleFunc("GET /api/projects/{id}/logs/history", docker.HandleProjectLogsHistory())
	apiMux.HandleFunc("/api/settings", docker.HandleSettings(database))

	// Mount protected API multiplexer under /api/ with middleware
	mux.Handle("/api/", docker.AuthMiddleware(database, apiMux.ServeHTTP))

	// Path-based application hosting route: http://140.245.116.79/app/<project-name>/
	mux.HandleFunc("/app/", docker.HandleAppProxy(dockClient))

	// WebSocket endpoints for real-time telemetry (Protected)
	mux.HandleFunc("/ws/stats", docker.AuthMiddleware(database, docker.HandleStatsWS(dockClient)))
	mux.HandleFunc("/ws/logs", docker.AuthMiddleware(database, docker.HandleLogsWS(dockClient)))

	// --- 4. Embedded Svelte Frontend SPA Handler -----------------------------
	uiContent := ui.FS()
	fileServer := http.FileServer(http.FS(uiContent))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Do not return SPA 200.html fallback for API or WebSocket endpoints,
		// BUT first check if the API request originated from a hosted app.
		// If so, redirect it to the app's proxy subpath.
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") || r.URL.Path == "/ask" || r.URL.Path == "/healthz" {
			if strings.HasPrefix(r.URL.Path, "/api") {
				referer := r.Header.Get("Referer")
				if referer != "" {
					refParts := strings.Split(referer, "/app/")
					if len(refParts) > 1 {
						projectName := strings.Split(refParts[1], "/")[0]
						if projectName != "" {
							// Redirect the /api/* call to the app's proxy so the backend container handles it
							http.Redirect(w, r, "/app/"+projectName+r.URL.RequestURI(), http.StatusTemporaryRedirect)
							return
						}
					}
				}
			}
			http.NotFound(w, r)
			return
		}

		// Clean and sanitize requested URL path
		cleanedPath := path.Clean(r.URL.Path)
		if strings.HasPrefix(cleanedPath, "/") {
			cleanedPath = cleanedPath[1:]
		}
		if cleanedPath == "" {
			cleanedPath = "index.html"
		}

		// Check if the requested file exists in embedded FS
		if _, err := fs.Stat(uiContent, cleanedPath); err != nil {
			// If not found in DevPanel UI, check if this is an asset requested by a hosted app
			referer := r.Header.Get("Referer")
			if referer != "" {
				parts := strings.Split(referer, "/app/")
				if len(parts) > 1 {
					projectName := strings.Split(parts[1], "/")[0]
					if projectName != "" {
						// Redirect root asset to the app's subpath
						http.Redirect(w, r, "/app/"+projectName+r.URL.Path, http.StatusTemporaryRedirect)
						return
					}
				}
			}

			// Fallback to SPA 200.html for client-side routing
			r.URL.Path = "/200.html"
		}

		fileServer.ServeHTTP(w, r)
	})

	// Wrap root multiplexer with request tracking middleware
	handler := tracker.Middleware(mux)

	// --- Server Configuration -----------------------------------------------
	// Note: Avoid setting WriteTimeout on the global server when streaming WebSockets,
	// as WriteTimeout applies to the entire connection duration.
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// --- 3. Signal & Shutdown Lifecycle --------------------------------------
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	serverErrCh := make(chan error, 1)

	// --- 2. Serve HTTP Requests ----------------------------------------------
	go func() {
		log.Println("main: starting HTTP server serve loop")
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	// Await termination trigger (OS Signal, Idle Timeout Context Cancellation, or Serve Error)
	select {
	case sig := <-sigCh:
		log.Printf("main: received signal %s — triggering graceful shutdown", sig)
		cancel()
	case <-ctx.Done():
		log.Println("main: context cancelled (idle timeout or shutdown signal)")
	case err := <-serverErrCh:
		if err != nil {
			log.Printf("main: HTTP server unexpected failure: %v", err)
		}
		cancel()
	}

	// Stop signal listening to release OS handlers
	signal.Stop(sigCh)
	idle.Stop()

	// Execute orderly graceful shutdown
	log.Println("main: performing graceful HTTP server shutdown...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("main: error during server shutdown: %v", err)
	}

	log.Println("main: shutdown sequence completed cleanly")
}
