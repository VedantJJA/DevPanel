package docker

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// HandleAppProxy handles routing requests to http://140.245.116.79/app/<project>/*
func HandleAppProxy(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Path format: /app/<project-name>/...
		path := strings.TrimPrefix(r.URL.Path, "/app/")
		parts := strings.SplitN(path, "/", 2)

		projectName := parts[0]
		if projectName == "" {
			http.Error(w, "App project name is required", http.StatusBadRequest)
			return
		}

		// Enforce trailing slash on project root to fix relative asset paths (e.g. ./assets/js)
		if len(parts) == 1 && !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, "/app/"+projectName+"/", http.StatusMovedPermanently)
			return
		}

		// Find running container matching devpnl-<project>-*
		containers, err := dockClient.ListContainers(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list containers: %v", err), http.StatusInternalServerError)
			return
		}

		var targetPort uint16
		var found bool
		var bestScore int = -1

		// Rewrite URL path so container receives root path
		subPath := "/"
		if len(parts) > 1 {
			subPath = "/" + parts[1]
		}
		
		isAPIRequest := strings.HasPrefix(subPath, "/api/") || subPath == "/api"

		expectedPrefix := fmt.Sprintf("devpnl-%s-", projectName)
		exactName := fmt.Sprintf("devpnl-%s", projectName)

		for _, c := range containers {
			cName := ""
			if len(c.Names) > 0 {
				cName = strings.TrimPrefix(c.Names[0], "/")
			}

			if strings.HasPrefix(cName, expectedPrefix) || cName == exactName {
				if c.State == "running" && len(c.Ports) > 0 {
					var port uint16
					for _, p := range c.Ports {
						if p.PublicPort > 0 {
							port = p.PublicPort
							break
						}
					}

					if port > 0 {
						score := 0
						lowerName := strings.ToLower(cName)
						svcType := c.Labels["devpanel.service.type"]
						
						if isAPIRequest {
							// For API requests, prioritize backend/api/web
							if svcType == "web" || svcType == "api" || svcType == "backend" {
								score = 100
							} else if strings.Contains(lowerName, "api") || strings.Contains(lowerName, "backend") {
								score = 100
							} else if strings.HasSuffix(lowerName, "-web") || strings.Contains(lowerName, "app") {
								score = 50
							} else if svcType == "static" || strings.HasSuffix(lowerName, "-frontend") || strings.Contains(lowerName, "-ui") {
								score = 10 // Last resort for API
							} else if svcType == "database" || strings.Contains(lowerName, "db") || strings.Contains(lowerName, "database") || strings.Contains(lowerName, "redis") || strings.Contains(lowerName, "postgres") {
								score = -100
							} else {
								score = 5
							}
						} else {
							// For UI requests, prioritize frontend/static
							if svcType == "static" || strings.HasSuffix(lowerName, "-frontend") || strings.Contains(lowerName, "-ui") {
								score = 100
							} else if svcType == "web" || strings.HasSuffix(lowerName, "-web") || strings.Contains(lowerName, "app") {
								score = 50
							} else if svcType == "api" || svcType == "backend" || strings.Contains(lowerName, "api") || strings.Contains(lowerName, "backend") {
								score = 10 
							} else if svcType == "database" || strings.Contains(lowerName, "db") || strings.Contains(lowerName, "database") || strings.Contains(lowerName, "redis") || strings.Contains(lowerName, "postgres") {
								score = -100
							} else {
								score = 5
							}
						}

						if score > bestScore {
							bestScore = score
							targetPort = port
							found = true
						}
					}
				}
			}
		}

		if !found || targetPort == 0 {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>DevPanel - App Not Found</title></head>
<body style="background:#09090b;color:#f4f4f5;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;">
  <div style="text-align:center;max-width:400px;padding:32px;background:#18181b;border:1px solid #27272a;border-radius:16px;">
    <h2 style="color:#ef4444;margin-top:0;">404 - App Not Found</h2>
    <p style="color:#a1a1aa;font-size:14px;">No active running container found for <b>%s</b>.</p>
    <a href="/" style="color:#10b981;text-decoration:none;font-weight:bold;font-size:14px;">← Back to DevPanel Dashboard</a>
  </div>
</body>
</html>`, projectName)
			return
		}

		targetURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", targetPort))
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid target URL: %v", err), http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)



		r.URL.Path = subPath
		r.URL.RawPath = subPath
		r.Header.Set("X-Forwarded-Host", r.Host)
		r.Header.Set("X-Forwarded-Prefix", fmt.Sprintf("/app/%s", projectName))

		log.Printf("proxy: forwarding /app/%s%s → %s", projectName, subPath, targetURL.String())
		proxy.ServeHTTP(w, r)
	}
}
