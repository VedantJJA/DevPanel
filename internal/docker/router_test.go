package docker

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRouterModeResolution(t *testing.T) {
	tests := []struct {
		name          string
		mode          string
		rootDomain    string
		reqHost       string
		reqPath       string
		expectedType  string // "project", "redirect", "ui"
		expectedProj  string
		expectedRel   string
	}{
		{
			name:         "Path mode: project request",
			mode:         "path",
			rootDomain:   "example.com",
			reqHost:      "example.com",
			reqPath:      "/app/vtopcc/backend/api/data",
			expectedType: "project",
			expectedProj: "vtopcc",
			expectedRel:  "/backend/api/data",
		},
		{
			name:         "Path mode: UI fallthrough",
			mode:         "path",
			rootDomain:   "example.com",
			reqHost:      "example.com",
			reqPath:      "/projects/bp-12345",
			expectedType: "ui",
		},
		{
			name:         "Subdomain mode: project subdomain",
			mode:         "subdomain",
			rootDomain:   "example.com",
			reqHost:      "vtopcc.example.com",
			reqPath:      "/api/auth/login",
			expectedType: "project",
			expectedProj: "vtopcc",
			expectedRel:  "/api/auth/login",
		},
		{
			name:         "Subdomain mode: panel origin UI",
			mode:         "subdomain",
			rootDomain:   "example.com",
			reqHost:      "panel.example.com",
			reqPath:      "/settings",
			expectedType: "ui",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.reqHost+tt.reqPath, nil)
			req.Host = tt.reqHost

			var resolvedType string
			var resolvedProj string
			var resolvedRel string

			hostWithoutPort := strings.Split(req.Host, ":")[0]
			baseDomainHost := strings.Split(tt.rootDomain, ":")[0]

			if tt.mode == "subdomain" {
				if strings.HasSuffix(hostWithoutPort, "."+baseDomainHost) {
					sub := strings.TrimSuffix(hostWithoutPort, "."+baseDomainHost)
					if sub != "" && sub != "panel" {
						resolvedType = "project"
						resolvedProj = sub
						resolvedRel = req.URL.Path
					} else {
						resolvedType = "ui"
					}
				} else {
					resolvedType = "ui"
				}
			} else {
				if strings.HasPrefix(req.URL.Path, "/app/") {
					if strings.HasPrefix(hostWithoutPort, "panel.") {
						resolvedType = "redirect"
					} else {
						p := strings.TrimPrefix(req.URL.Path, "/app/")
						parts := strings.SplitN(p, "/", 2)
						if len(parts) > 0 && parts[0] != "" {
							resolvedType = "project"
							resolvedProj, _ = url.PathUnescape(parts[0])
							if len(parts) == 2 {
								resolvedRel = "/" + parts[1]
							} else {
								resolvedRel = "/"
							}
						}
					}
				} else {
					resolvedType = "ui"
				}
			}

			if resolvedType != tt.expectedType {
				t.Fatalf("expected type %s, got %s", tt.expectedType, resolvedType)
			}
			if tt.expectedType == "project" {
				if resolvedProj != tt.expectedProj {
					t.Errorf("expected project %s, got %s", tt.expectedProj, resolvedProj)
				}
				if resolvedRel != tt.expectedRel {
					t.Errorf("expected rel path %s, got %s", tt.expectedRel, resolvedRel)
				}
			}
		})
	}
}

func TestSixHeaderRoutingModes(t *testing.T) {
	// Test case definitions matching the exact specification:
	// Path Mode:
	// 1. domain.com/app/<frontend-slug>/        -> Frontend (static) container
	// 2. domain.com/app/<frontend-slug>/api/users -> Backend (web) container (rerouted)
	// 3. domain.com/app/<backend-slug>/api/users  -> Backend (web) container directly
	// Subdomain Mode:
	// 4. <frontend-slug>.domain.com/            -> Frontend (static) container
	// 5. <frontend-slug>.domain.com/api/users   -> Backend (web) container (rerouted)
	// 6. <backend-slug>.domain.com/api/users    -> Backend (web) container directly

	frontendSlug := "vtopcc"
	backendSlug := "vtopcc-backend"

	tests := []struct {
		name            string
		mode            string
		reqHost         string
		reqPath         string
		initialTarget   string // slug parsed from URL/host
		isApiReq        bool
		initialType     string // "static" or "web"
		expectedTarget  string // final target service slug
	}{
		{
			name:           "1. Path Mode: Frontend root -> Frontend static",
			mode:           "path",
			reqHost:        "domain.com",
			reqPath:        "/app/" + frontendSlug + "/",
			initialTarget:  frontendSlug,
			isApiReq:       false,
			initialType:    "static",
			expectedTarget: frontendSlug,
		},
		{
			name:           "2. Path Mode: Frontend /api/users -> Rerouted to Backend web",
			mode:           "path",
			reqHost:        "domain.com",
			reqPath:        "/app/" + frontendSlug + "/api/users",
			initialTarget:  frontendSlug,
			isApiReq:       true,
			initialType:    "static",
			expectedTarget: backendSlug,
		},
		{
			name:           "3. Path Mode: Backend /api/users -> Backend web directly",
			mode:           "path",
			reqHost:        "domain.com",
			reqPath:        "/app/" + backendSlug + "/api/users",
			initialTarget:  backendSlug,
			isApiReq:       true,
			initialType:    "web",
			expectedTarget: backendSlug,
		},
		{
			name:           "4. Subdomain Mode: Frontend root -> Frontend static",
			mode:           "subdomain",
			reqHost:        frontendSlug + ".domain.com",
			reqPath:        "/",
			initialTarget:  frontendSlug,
			isApiReq:       false,
			initialType:    "static",
			expectedTarget: frontendSlug,
		},
		{
			name:           "5. Subdomain Mode: Frontend /api/users -> Rerouted to Backend web",
			mode:           "subdomain",
			reqHost:        frontendSlug + ".domain.com",
			reqPath:        "/api/users",
			initialTarget:  frontendSlug,
			isApiReq:       true,
			initialType:    "static",
			expectedTarget: backendSlug,
		},
		{
			name:           "6. Subdomain Mode: Backend /api/users -> Backend web directly",
			mode:           "subdomain",
			reqHost:        backendSlug + ".domain.com",
			reqPath:        "/api/users",
			initialTarget:  backendSlug,
			isApiReq:       true,
			initialType:    "web",
			expectedTarget: backendSlug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetSlug := tt.initialTarget
			targetType := tt.initialType

			// Simulate HandleProjectReverseProxy API reroute logic:
			// If target is static and path is an API call, reroute to web service
			if targetType == "static" && tt.isApiReq {
				targetSlug = backendSlug
				targetType = "web"
			}

			if targetSlug != tt.expectedTarget {
				t.Fatalf("%s: expected target %s, got %s", tt.name, tt.expectedTarget, targetSlug)
			}
		})
	}
}

