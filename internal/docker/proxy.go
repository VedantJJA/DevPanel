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
const XDevPanelProject = "X-Devpanel-Project"

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
	for i := 0; i < 3; i++ {
		resp, err = base.RoundTrip(req)
		if err == nil {
			return resp, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return resp, err
}

// findContainerPort finds the public host port for a container belonging to a project service.
func findContainerPort(containers []ContainerSummary, bpID, bpName, serviceName string) int {
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
			for _, t := range targets {
				if cleanName == t {
					for _, p := range c.Ports {
						if p.PublicPort > 0 {
							return int(p.PublicPort)
						}
					}
				}
			}
		}
	}
	return 0
}

// HandleProjectReverseProxy handles routing for /app/{project}/{service}/ and subdomain routing (*.domain.com, *.nip.io)
func HandleProjectReverseProxy(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse project and service names from either URL Path or Subdomain Host header
		host := strings.Split(r.Host, ":")[0]
		subParts := strings.Split(host, ".")

		projectName := ""
		serviceName := ""

		// 1. Try URL Path /app/{project}/{service}/
		if strings.HasPrefix(r.URL.Path, "/app/") {
			p := strings.TrimPrefix(r.URL.Path, "/app/")
			parts := strings.Split(strings.Trim(p, "/"), "/")
			if len(parts) > 0 && parts[0] != "" {
				projectName = parts[0]
				if len(parts) >= 2 {
					serviceName = parts[1]
				}
			}
		}

		// 2. Try Subdomain / Domain Resolution if path /app/ was not used
		if projectName == "" && len(subParts) >= 2 {
			first := strings.ToLower(subParts[0])
			if first != "localhost" && first != "127" && first != "www" && first != "devpanel" && first != "panel" {
				// Pattern A: Check if first subdomain matches a project (e.g. vtopcc.nip.io or vtopcc.domain.com)
				bpCheck, err := database.GetBlueprint(r.Context(), first)
				if err == nil && bpCheck != nil {
					projectName = bpCheck.ID
				} else if len(subParts) >= 3 {
					// Pattern B: Check if second subdomain matches a project (e.g. vtopcc-backend.vtopcc.nip.io)
					second := strings.ToLower(subParts[1])
					bpCheck2, err2 := database.GetBlueprint(r.Context(), second)
					if err2 == nil && bpCheck2 != nil {
						projectName = bpCheck2.ID
						serviceName = first
					}
				}

				// Pattern C: Check if full Host or first subdomain matches a service or custom domain
				if projectName == "" {
					svcCheck, _ := database.FindServiceByName(r.Context(), host)
					if svcCheck == nil {
						svcCheck, _ = database.FindServiceByName(r.Context(), first)
					}
					if svcCheck != nil {
						projectName = svcCheck.ProjectID
						serviceName = svcCheck.Name
					}
				}
			}
		}

		// Fallback: project injected by rootRouter via referer/cookie dispatch.
		if projectName == "" {
			if injected := r.Header.Get(XDevPanelProject); injected != "" {
				projectName = injected
			}
		}
		// Always strip the internal header before forwarding to the upstream.
		r.Header.Del(XDevPanelProject)

		if projectName != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "devpanel_project",
				Value:    projectName,
				Path:     "/",
				SameSite: http.SameSiteLaxMode,
			})
		}

		if projectName == "" {
			http.Error(w, "Project or service not found for host/path", http.StatusBadRequest)
			return
		}

		// Find blueprint / services for project
		bp, err := database.GetBlueprint(r.Context(), projectName)
		if err != nil || bp == nil {
			http.Error(w, fmt.Sprintf("Project %q not found", projectName), http.StatusNotFound)
			return
		}

		svcs, err := database.ListServices(r.Context(), bp.ID)
		if err != nil || len(svcs) == 0 {
			http.Error(w, fmt.Sprintf("No active services for project %q", projectName), http.StatusNotFound)
			return
		}

		var targetSvc *db.ServiceRecord
		if serviceName != "" {
			for i := range svcs {
				if strings.EqualFold(svcs[i].Name, serviceName) {
					targetSvc = &svcs[i]
					break
				}
			}
			// If serviceName was extracted from path (e.g. /app/project/api/...) but does not match
			// any actual service name in the project, clear serviceName so it isn't treated as a sub-service.
			if targetSvc == nil {
				serviceName = ""
			}
		}

		// Intelligent routing fallback when serviceName is omitted or un-matched:
		// If request path is an API call (/api/*, /app/<project>/api/*, etc.), route to "web" service (backend API)!
		isApiReq := strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" ||
			strings.Contains(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, fmt.Sprintf("/app/%s/api", projectName))

		if targetSvc == nil && isApiReq {
			for i := range svcs {
				if svcs[i].Type == "web" {
					targetSvc = &svcs[i]
					break
				}
			}
		}

		// Default to static (frontend) service first, then web service
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
		containerPort := findContainerPort(containers, bp.ID, bp.Name, targetSvc.Name)

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

		// Dynamically rewrite HTML base and asset URLs to support subpath hosting (/app/<project>/)
		proxy.ModifyResponse = func(resp *http.Response) error {
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "text/html") && resp.Body != nil {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil
				}
				resp.Body.Close()

				htmlStr := string(bodyBytes)

				subpath := fmt.Sprintf("/app/%s/", projectName)
				if serviceName != "" {
					subpath = fmt.Sprintf("/app/%s/%s/", projectName, serviceName)
				}

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
<html>
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

		// Strip /app/{project}/{service} or /app/{project} prefix from request path if present
		if strings.HasPrefix(r.URL.Path, "/app/") {
			var relPath string
			if serviceName != "" {
				prefixWithSvc := fmt.Sprintf("/app/%s/%s", projectName, serviceName)
				if strings.HasPrefix(r.URL.Path, prefixWithSvc) {
					relPath = strings.TrimPrefix(r.URL.Path, prefixWithSvc)
				}
			}
			if relPath == "" {
				prefixBase := fmt.Sprintf("/app/%s", projectName)
				relPath = strings.TrimPrefix(r.URL.Path, prefixBase)
			}
			if relPath == "" || !strings.HasPrefix(relPath, "/") {
				relPath = "/" + relPath
			}
			r.URL.Path = relPath
		}

		proxy.ServeHTTP(w, r)
	}
}
