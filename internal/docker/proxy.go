package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/VedantJJA/devpnl/internal/db"
)

// XDevPanelProject is an internal request header set by rootRouter when dispatching
// a request via Referer-based routing. HandleProjectReverseProxy reads it to
// identify the target project when the URL path doesn't start with /app/<project>.
const (
	XDevPanelProject     = "X-Devpanel-Project"
	XDevPanelRoutingMode = "X-Devpanel-Routing-Mode"
)

// retryTransport retries HTTP requests when encountering TCP connection resets during container boot.
type retryTransport struct {
	base http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	var bodyBytes []byte
	if req.Body != nil {
		var readErr error
		bodyBytes, readErr = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
	}

	for i := 0; i < 3; i++ {
		reqCopy := req.Clone(req.Context())
		if bodyBytes != nil {
			reqCopy.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			reqCopy.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
		}

		resp, err = base.RoundTrip(reqCopy)
		if err == nil {
			return resp, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return resp, err
}

// findContainerPort finds the public host port for a container belonging to a project service.
func findContainerPort(containers []ContainerSummary, bpID, bpName, serviceName string, targetPort int) int {
	if serviceName == "" {
		return 0
	}
	sName := strings.ToLower(serviceName)
	cleanID := strings.ToLower(strings.TrimPrefix(bpID, "bp-"))
	pName := strings.ToLower(bpName)

	targets := []string{
		fmt.Sprintf("devpnl-%s-%s", cleanID, sName),
		fmt.Sprintf("devpnl-%s-%s", strings.ToLower(bpID), sName),
	}
	if pName != "" {
		targets = append(targets, fmt.Sprintf("devpnl-%s-%s", pName, sName))
	}

	for _, c := range containers {
		for _, n := range c.Names {
			cleanName := strings.ToLower(strings.TrimPrefix(n, "/"))
			matched := false
			for _, t := range targets {
				if cleanName == t {
					matched = true
					break
				}
			}
			if !matched && (strings.HasSuffix(cleanName, "-"+sName) || cleanName == sName) {
				matched = true
			}
			if matched {
				// 1. Prefer port matching the service's target private port
				if targetPort > 0 {
					for _, p := range c.Ports {
						if int(p.PrivatePort) == targetPort && p.PublicPort > 0 {
							return int(p.PublicPort)
						}
					}
				}
				// 2. Fallback to any public port if exact private port match is missing
				for _, p := range c.Ports {
					if p.PublicPort > 0 {
						return int(p.PublicPort)
					}
				}
			}
		}
	}
	return 0
}

// HandleProjectReverseProxy handles routing for /app/{slug}/ and subdomain routing ({slug}.domain.com).
// It resolves the target service by slug (Render-style), and automatically reroutes /api/* requests
// from a static/frontend service to the project's web/backend service container.
func HandleProjectReverseProxy(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse service slug from either URL Path or Subdomain Host header
		host := strings.Split(r.Host, ":")[0]
		subParts := strings.Split(host, ".")

		resolvedSlug := ""
		serviceName := ""
		projectName := ""

		// 1. Try URL Path /app/{slug}/...
		if strings.HasPrefix(r.URL.Path, "/app/") {
			p := strings.TrimPrefix(r.URL.Path, "/app/")
			parts := strings.Split(strings.Trim(p, "/"), "/")
			if len(parts) > 0 && parts[0] != "" {
				resolvedSlug = parts[0]
			}
		}

		// 2. Try Subdomain resolution: {slug}.domain.com or {slug}.nip.io
		if resolvedSlug == "" && len(subParts) >= 2 {
			first := strings.ToLower(subParts[0])
			if first != "localhost" && first != "127" && first != "www" && first != "devpanel" && first != "panel" {
				resolvedSlug = first
			}
		}

		// 3. Fallback: project/slug injected by rootRouter via referer/cookie dispatch
		if resolvedSlug == "" {
			if injected := r.Header.Get(XDevPanelProject); injected != "" {
				resolvedSlug = injected
			}
		}
		currentRoutingMode := r.Header.Get(XDevPanelRoutingMode)
		// Always strip internal headers before forwarding to the upstream.
		r.Header.Del(XDevPanelProject)
		r.Header.Del(XDevPanelRoutingMode)

		if resolvedSlug == "" {
			http.Error(w, "Project or service not found for host/path", http.StatusBadRequest)
			return
		}

		// --- Resolve service by slug (Render-style) ---
		// First try: exact slug match on the services table
		svcBySlug, _ := database.FindServiceBySlug(r.Context(), resolvedSlug)

		var bp *db.BlueprintRecord
		var svcs []db.ServiceRecord
		var targetSvc *db.ServiceRecord

		if svcBySlug != nil {
			// Slug matched a specific service
			projectName = svcBySlug.ProjectID
			serviceName = svcBySlug.Name

			var err error
			bp, err = database.GetBlueprint(r.Context(), projectName)
			if err != nil || bp == nil {
				http.Error(w, fmt.Sprintf("Project not found for service slug %q", resolvedSlug), http.StatusNotFound)
				return
			}
			svcs, _ = database.ListServices(r.Context(), bp.ID)
			targetSvc = svcBySlug
		} else {
			// Fallback: try resolving as project name/ID (backward compat)
			var err error
			bp, err = database.GetBlueprint(r.Context(), resolvedSlug)
			if err != nil || bp == nil {
				// Also check by service name (legacy lookup)
				if svcCheck, _ := database.FindServiceByName(r.Context(), resolvedSlug); svcCheck != nil {
					projectName = svcCheck.ProjectID
					serviceName = svcCheck.Name
					bp, err = database.GetBlueprint(r.Context(), projectName)
					if err != nil || bp == nil {
						http.Error(w, fmt.Sprintf("Project or service %q not found", resolvedSlug), http.StatusNotFound)
						return
					}
				} else {
					http.Error(w, fmt.Sprintf("Project or service %q not found", resolvedSlug), http.StatusNotFound)
					return
				}
			} else {
				projectName = bp.ID
			}

			svcs, _ = database.ListServices(r.Context(), bp.ID)
			if len(svcs) == 0 {
				http.Error(w, fmt.Sprintf("No active services for project %q", projectName), http.StatusNotFound)
				return
			}

			// Try matching service name from path or legacy
			if serviceName != "" {
				for i := range svcs {
					if strings.EqualFold(svcs[i].Name, serviceName) || strings.EqualFold(svcs[i].Slug, serviceName) {
						targetSvc = &svcs[i]
						break
					}
				}
			}

			// Default to static (frontend) first, then web
			if targetSvc == nil {
				for i := range svcs {
					if svcs[i].Type == "static" {
						targetSvc = &svcs[i]
						break
					}
				}
			}
			if targetSvc == nil {
				for i := range svcs {
					if svcs[i].Type == "web" {
						targetSvc = &svcs[i]
						break
					}
				}
			}
			if targetSvc == nil {
				targetSvc = &svcs[0]
			}
		}

		// Set cookie for referer-based routing fallback (scoped to /app/<slug>/)
		if resolvedSlug != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "devpanel_project",
				Value:    resolvedSlug,
				Path:     fmt.Sprintf("/app/%s/", resolvedSlug),
				SameSite: http.SameSiteLaxMode,
			})
		}

		// --- Frontend→Backend API reroute ---
		// If the target service is a "static" (frontend) type, and the request path
		// is an API call (/api/*), automatically reroute to the project's "web" (backend) service.
		isApiReq := false
		requestPath := r.URL.Path
		if strings.HasPrefix(requestPath, "/app/") {
			// Strip /app/{slug}/ prefix to get the relative path
			stripped := strings.TrimPrefix(requestPath, "/app/")
			parts := strings.SplitN(stripped, "/", 2)
			if len(parts) == 2 {
				requestPath = "/" + parts[1]
			} else {
				requestPath = "/"
			}
		}
		isApiReq = strings.HasPrefix(requestPath, "/api/") || requestPath == "/api"

		if targetSvc != nil && targetSvc.Type == "static" && isApiReq {
			// Find the web/backend service in the same project
			for i := range svcs {
				if svcs[i].Type == "web" {
					targetSvc = &svcs[i]
					serviceName = svcs[i].Name
					break
				}
			}
		}

		// Even if we didn't match via slug initially and matched via old project lookup,
		// check for /api/* rerouting on any non-backend service
		if targetSvc != nil && targetSvc.Type != "web" && isApiReq {
			for i := range svcs {
				if svcs[i].Type == "web" {
					targetSvc = &svcs[i]
					serviceName = svcs[i].Name
					break
				}
			}
		}

		targetPort := targetSvc.Port
		if targetPort <= 0 {
			if targetSvc.Type == "static" {
				targetPort = 80
			} else {
				targetPort = 8080
			}
		}

		// Check if container is running via Docker API
		containers, _ := dockClient.ListContainers(r.Context())
		containerPort := findContainerPort(containers, bp.ID, bp.Name, targetSvc.Name, targetPort)

		if containerPort == 0 {
			containerPort = targetPort
		}

		targetURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", containerPort))
		if err != nil {
			http.Error(w, "Invalid proxy target", http.StatusInternalServerError)
			return
		}

		// Reverse proxy handler with retry transport for container boot resilience
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.Transport = &retryTransport{
			base: &http.Transport{
				ResponseHeaderTimeout: 15 * time.Second,
			},
		}

		// Use the resolved slug for subpath-based HTML rewriting
		proxySlug := resolvedSlug

		// Dynamically rewrite HTML base and asset URLs to support subpath hosting (/app/{slug}/)
		proxy.ModifyResponse = func(resp *http.Response) error {
			if currentRoutingMode == "subdomain" {
				return nil
			}
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "text/html") && resp.Body != nil {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil
				}
				resp.Body.Close()

				htmlStr := string(bodyBytes)

				subpath := fmt.Sprintf("/app/%s/", proxySlug)

				if !strings.Contains(htmlStr, "<base ") && !strings.Contains(htmlStr, "<BASE ") {
					baseTag := fmt.Sprintf(`<head><base href="%s">`, subpath)
					if strings.Contains(htmlStr, "<head>") {
						htmlStr = strings.Replace(htmlStr, "<head>", baseTag, 1)
					} else if strings.Contains(htmlStr, "<HEAD>") {
						htmlStr = strings.Replace(htmlStr, "<HEAD>", baseTag, 1)
					} else if strings.Contains(htmlStr, "<html>") {
						htmlStr = strings.Replace(htmlStr, "<html>", "<html>"+baseTag, 1)
					}
				}

				htmlStr = strings.ReplaceAll(htmlStr, `src="/assets/`, `src="assets/`)
				htmlStr = strings.ReplaceAll(htmlStr, `href="/assets/`, `href="assets/`)
				htmlStr = strings.ReplaceAll(htmlStr, `src="/static/`, `src="static/`)
				htmlStr = strings.ReplaceAll(htmlStr, `href="/static/`, `href="static/`)

				newBody := []byte(htmlStr)
				resp.Body = io.NopCloser(bytes.NewReader(newBody))
				resp.ContentLength = int64(len(newBody))
				resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
			}
			return nil
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
			isAPI := strings.Contains(req.URL.Path, "/api") ||
				strings.Contains(req.Header.Get("Accept"), "application/json") ||
				req.Header.Get("X-Requested-With") == "XMLHttpRequest"

			w.WriteHeader(http.StatusBadGateway)

			if isAPI {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"error":      fmt.Sprintf("Service container %q is unreachable: %v", targetSvc.Name, err),
					"service":    targetSvc.Name,
					"project":    projectName,
					"backendUrl": fmt.Sprintf("http://localhost:%d", containerPort),
				})
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, `<!DOCTYPE html>
					<head>
						<title>%s — DevPanel App Unavailable</title>
						<meta charset="utf-8">
						<style>
							body { font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
							.card { background: #1e293b; border: 1px solid #334155; padding: 2.5rem; border-radius: 1rem; max-width: 500px; text-align: center; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.5); }
							.badge { display: inline-block; background: #ef4444; color: white; font-weight: bold; font-size: 0.75rem; padding: 0.25rem 0.75rem; border-radius: 9999px; text-transform: uppercase; margin-bottom: 1rem; }
							h1 { margin: 0 0 0.5rem 0; font-size: 1.5rem; }
							p { color: #94a3b8; font-size: 0.9rem; line-height: 1.5; margin-bottom: 1.5rem; }
							.code { background: #0f172a; border: 1px solid #334155; padding: 0.75rem; border-radius: 0.5rem; font-family: monospace; font-size: 0.85rem; color: #f87171; word-break: break-all; }
						</style>
					</head>
					<body>
						<div class="card">
							<div class="badge">Service Offline</div>
							<h1>%s / %s</h1>
							<p>Service container is starting up or temporarily unreachable on host port %d.</p>
							<div class="code">%v</div>
						</div>
					</body>
				</html>`, targetSvc.Name, projectName, targetSvc.Name, containerPort, err)
			}
		}

		// Strip /app/{slug} prefix from request path if present, so upstream sees relative paths
		if strings.HasPrefix(r.URL.Path, "/app/") {
			stripped := strings.TrimPrefix(r.URL.Path, "/app/")
			parts := strings.SplitN(stripped, "/", 2)
			relPath := "/"
			if len(parts) == 2 {
				relPath = "/" + parts[1]
			}
			r.URL.Path = relPath
		}

		proxy.ServeHTTP(w, r)
	}
}

