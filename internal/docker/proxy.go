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

	"github.com/VedantJJA/devpnl/internal/db"
)

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

// HandleProjectReverseProxy handles routing for /app/{project}/{service}/ and /app/{project}/
func HandleProjectReverseProxy(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse project and service names from either URL Path or Subdomain Host header
		host := strings.Split(r.Host, ":")[0]
		subParts := strings.Split(host, ".")

		projectName := ""
		serviceName := ""

		if strings.HasPrefix(r.URL.Path, "/app/") {
			p := strings.TrimPrefix(r.URL.Path, "/app/")
			parts := strings.Split(strings.Trim(p, "/"), "/")
			if len(parts) > 0 && parts[0] != "" {
				projectName = parts[0]
				if len(parts) >= 2 {
					serviceName = parts[1]
				}
			}
		} else if len(subParts) >= 2 && subParts[0] != "localhost" && subParts[0] != "127" && !strings.Contains(host, "5173") {
			// Subdomain format: <service>.<project>.<domain> or <project>.<domain>
			if len(subParts) >= 3 {
				serviceName = subParts[0]
				projectName = subParts[1]
			} else {
				projectName = subParts[0]
			}
		}

		if projectName == "" {
			http.Error(w, "Project name required in path /app/<project> or subdomain", http.StatusBadRequest)
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

		// Reverse proxy handler
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

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

		// Strip /app/{project}/{service} or /app/{project} prefix from request path
		prefixWithSvc := fmt.Sprintf("/app/%s/%s", projectName, targetSvc.Name)
		prefixBase := fmt.Sprintf("/app/%s", projectName)
		var relPath string
		if strings.HasPrefix(r.URL.Path, prefixWithSvc) {
			relPath = strings.TrimPrefix(r.URL.Path, prefixWithSvc)
		} else {
			relPath = strings.TrimPrefix(r.URL.Path, prefixBase)
		}
		if relPath == "" || !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}
		r.URL.Path = relPath
		proxy.ServeHTTP(w, r)
	}
}
