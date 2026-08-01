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
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
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

	// Synchronize Nginx dynamic subdomains on startup
	if err := docker.SyncNginx(dockClient); err != nil {
		log.Printf("main: failed to sync nginx routes on startup: %v", err)
	}

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
	mux.HandleFunc("POST /api/auth/logout", docker.HandleAuthLogout(database))

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
	apiMux.HandleFunc("GET /api/projects/{id}", docker.HandleGetProject(database, dockClient))
	apiMux.HandleFunc("PATCH /api/projects/{id}/services/{name}", docker.HandleUpdateService(database))
	apiMux.HandleFunc("POST /api/projects/{id}/deploy", docker.HandleTriggerProjectDeploy(database, dockClient))
	apiMux.HandleFunc("GET /api/projects/{id}/deployments", docker.HandleListDeployments(database))
	apiMux.HandleFunc("GET /api/projects/{id}/logs", docker.HandleProjectLogsSSE())
	apiMux.HandleFunc("POST /api/projects/{id}/services/{name}/restart", docker.HandleRestartService(database, dockClient))
	apiMux.HandleFunc("GET /api/projects/{id}/services/{name}/logs", docker.HandleServiceLogsSSE(dockClient))
	apiMux.HandleFunc("GET /api/projects/{id}/logs/history", docker.HandleProjectLogsHistory())
	apiMux.HandleFunc("DELETE /api/projects/{id}/logs", docker.HandleClearProjectLogs())
	apiMux.HandleFunc("DELETE /api/projects/{id}", docker.HandleDeleteProject(database, dockClient))
	apiMux.HandleFunc("/api/settings", docker.HandleSettings(database))

	// Mount protected API multiplexer under /api/
	apiHandler := docker.AuthMiddleware(database, apiMux.ServeHTTP)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		apiHandler.ServeHTTP(w, r)
	})

	// WebSocket endpoints for real-time telemetry (Protected)
	mux.HandleFunc("/ws/stats", docker.AuthMiddleware(database, docker.HandleStatsWS(dockClient)))
	mux.HandleFunc("/ws/logs", docker.AuthMiddleware(database, docker.HandleLogsWS(dockClient)))
	// Build-log WebSocket — streams in-process deploy logs with clear-on-new-build signalling
	mux.HandleFunc("/ws/projects/{id}/logs", docker.AuthMiddleware(database, docker.HandleBuildLogsWS()))

	// Reverse proxy route for /app/{project}/{service}/
	mux.HandleFunc("/app/", docker.HandleProjectReverseProxy(database, dockClient))

	// Embedded Svelte Frontend SPA Handler
	uiContent := ui.FS()
	fileServer := http.FileServer(http.FS(uiContent))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Do not return SPA 200.html fallback for API, reverse proxy, or WebSocket endpoints
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") || strings.HasPrefix(r.URL.Path, "/app") || r.URL.Path == "/ask" || r.URL.Path == "/healthz" {
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

	// Project Reverse Proxy Handler
	projectProxyHandler := docker.HandleProjectReverseProxy(database, dockClient)

	// Subdomain & Referer-aware root HTTP router.
	// Routing behaviour depends on the "routing_mode" setting (path | subdomain).
	rootRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routingMode, baseDomain := getRoutingMode(r.Context(), database, r)

		host := r.Host
		hostWithoutPort := strings.Split(host, ":")[0]
		baseDomainHost := strings.Split(baseDomain, ":")[0]
		firstSub := strings.ToLower(strings.Split(hostWithoutPort, ".")[0])

		isProjectSubdomain := hostWithoutPort != baseDomainHost &&
			hostWithoutPort != "panel."+baseDomainHost &&
			firstSub != "localhost" && firstSub != "127" &&
			firstSub != "panel" && firstSub != "devpanel" && firstSub != "www"

		// ── SUBDOMAIN ROUTING MODE ────────────────────────────────────────────
		if routingMode == "subdomain" {
			// Redirect /app/<project>[/...] → http(s)://<project>.<baseDomain>[/...]
			if strings.HasPrefix(r.URL.Path, "/app/") {
				rest := strings.TrimPrefix(r.URL.Path, "/app/")
				parts := strings.SplitN(rest, "/", 2)
				if len(parts) > 0 && parts[0] != "" {
					projectPart := parts[0]
					tail := ""
					if len(parts) == 2 {
						tail = "/" + parts[1]
					}
					scheme := "https"
					if strings.Contains(baseDomain, "localhost") || strings.Contains(baseDomain, "127.0.0.1") {
						scheme = "http"
					}
					portSuffix := ""
					if idx := strings.Index(baseDomain, ":"); idx >= 0 {
						portSuffix = baseDomain[idx:]
					}
					http.Redirect(w, r, fmt.Sprintf("%s://%s.%s%s%s", scheme, projectPart, baseDomainHost, portSuffix, tail), http.StatusFound)
					return
				}
			}

			// Route ALL requests on a project subdomain directly to that project container.
			// This includes /api/*, /ws/*, / etc. — no admin API intercept.
			if isProjectSubdomain {
				projectProxyHandler.ServeHTTP(w, r)
				return
			}

			// Fall through: serve DevPanel admin panel (panel.klouds.online itself)
			mux.ServeHTTP(w, r)
			return
		}

		// ── PATH ROUTING MODE (default) ───────────────────────────────────────

		// 1. Explicit /app/<project>/... path → project reverse proxy.
		if strings.HasPrefix(r.URL.Path, "/app/") {
			projectProxyHandler.ServeHTTP(w, r)
			return
		}

		// 2. Project subdomain in path mode (handles cross-origin XHR from the same domain).
		if isProjectSubdomain {
			projectProxyHandler.ServeHTTP(w, r)
			return
		}

		// 3. Referer & Cookie fallback: /app/<project>/-hosted SPA making absolute /api/* calls.
		//    Extract project name from Referer or devpanel_project Cookie, inject via header.
		if strings.HasPrefix(r.URL.Path, "/api/") && !isDevPanelAdminRoute(r.URL.Path) {
			projectName := ""

			referer := r.Header.Get("Referer")
			if idx := strings.Index(referer, "/app/"); idx >= 0 {
				rest := referer[idx+5:] // after "/app/"
				projectName = strings.SplitN(rest, "/", 2)[0]
			}

			if projectName == "" {
				if cookie, err := r.Cookie("devpanel_project"); err == nil && cookie.Value != "" {
					projectName = cookie.Value
				}
			}

			if projectName != "" {
				// Inject project name so HandleProjectReverseProxy can identify the target
				// without the path needing to start with /app/<project>.
				r2 := r.Clone(r.Context())
				r2.Header.Set(docker.XDevPanelProject, projectName)
				projectProxyHandler.ServeHTTP(w, r2)
				return
			}
		}

		// 4. Default: DevPanel Admin API / Web UI multiplexer.
		mux.ServeHTTP(w, r)
	})

	// Wrap root multiplexer with request tracking middleware
	handler := tracker.Middleware(rootRouter)

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

func isDevPanelAdminRoute(p string) bool {
	adminPrefixes := []string{
		"/api/auth/login",
		"/api/auth/me",
		"/api/auth/change-password",
		"/api/projects",
		"/api/repos",
		"/api/containers",
		"/api/deployments",
		"/api/blueprints",
		"/api/volumes",
		"/api/settings",
		"/api/stats",
		"/api/metrics",
		"/api/logs",
		"/api/system",
	}
	for _, prefix := range adminPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// routingModeCache caches the routing_mode and base_domain settings to avoid a
// DB round-trip on every HTTP request. The cache TTL is 30 seconds.
var (
	rmCache     string
	bdCache     string
	rmCacheAt   time.Time
	rmCacheMu   sync.RWMutex
)

func getRoutingMode(ctx context.Context, database *db.DB, r *http.Request) (mode, baseDomain string) {
	reqHost := ""
	if r != nil {
		reqHost = r.Header.Get("X-Forwarded-Host")
		if reqHost == "" {
			reqHost = r.Host
		}
		reqHost = strings.TrimPrefix(reqHost, "panel.")
	}

	rmCacheMu.RLock()
	if time.Since(rmCacheAt) < 30*time.Second {
		m, d := rmCache, bdCache
		rmCacheMu.RUnlock()
		if (d == "" || d == "localhost:8090") && reqHost != "" {
			d = reqHost
		}
		return m, d
	}
	rmCacheMu.RUnlock()

	rmCacheMu.Lock()
	defer rmCacheMu.Unlock()
	m, _ := database.GetSetting(ctx, "routing_mode")
	d, _ := database.GetSetting(ctx, "base_domain")
	if m == "" {
		m = "path"
	}
	if d == "" || d == "localhost:8090" {
		if reqHost != "" {
			d = reqHost
		} else {
			d = "localhost:8090"
		}
	}
	rmCache = m
	bdCache = d
	rmCacheAt = time.Now()
	return m, d
}
