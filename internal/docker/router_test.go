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
