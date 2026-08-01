package docker

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VedantJJA/devpnl/internal/db"
)

func TestProjectRequestResolution(t *testing.T) {
	errNotFound := errors.New("not found")

	mockGetBp := func(name string) (*db.BlueprintRecord, error) {
		if name == "my-project" || name == "bp-my-project" {
			return &db.BlueprintRecord{ID: "bp-my-project", Name: "my-project"}, nil
		}
		return nil, errNotFound
	}

	mockListSvcs := func(bpID string) ([]db.ServiceRecord, error) {
		if bpID == "bp-my-project" {
			return []db.ServiceRecord{
				{ID: 1, Name: "web", Type: "web", ProjectID: bpID},
			}, nil
		}
		return nil, errNotFound
	}

	t.Run("Subdomain Mode - Valid Project", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://my-project.example.com/dashboard", nil)
		target, statusCode, errStr := ResolveProjectRoute(req, "example.com", "subdomain", mockGetBp, mockListSvcs)

		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d (err: %s)", statusCode, errStr)
		}
		if target == nil || target.ProjectName != "my-project" {
			t.Fatalf("expected target project my-project, got %+v", target)
		}
		if target.Subpath != "/dashboard" {
			t.Fatalf("expected subpath /dashboard, got %s", target.Subpath)
		}
	})

	t.Run("Subdomain Mode - Panel Origin Fallthrough", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://panel.example.com/api/config", nil)
		target, statusCode, _ := ResolveProjectRoute(req, "example.com", "subdomain", mockGetBp, mockListSvcs)

		if target != nil || statusCode != 0 {
			t.Fatalf("expected fallthrough (target nil, status 0), got status %d", statusCode)
		}
	})

	t.Run("Subdomain Mode - Unknown Project 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://unknown-project.example.com/", nil)
		_, statusCode, _ := ResolveProjectRoute(req, "example.com", "subdomain", mockGetBp, mockListSvcs)

		if statusCode != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", statusCode)
		}
	})

	t.Run("Path Mode - Valid Project", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://example.com/app/my-project/api/health", nil)
		target, statusCode, errStr := ResolveProjectRoute(req, "example.com", "path", mockGetBp, mockListSvcs)

		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d (err: %s)", statusCode, errStr)
		}
		if target == nil || target.ProjectName != "my-project" {
			t.Fatalf("expected target project my-project, got %+v", target)
		}
		if target.Subpath != "/api/health" {
			t.Fatalf("expected subpath /api/health, got %s", target.Subpath)
		}
		if target.Prefix != "/app/my-project" {
			t.Fatalf("expected prefix /app/my-project, got %s", target.Prefix)
		}
	})

	t.Run("Path Mode - Unknown Project 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://example.com/app/nonexistent/api", nil)
		_, statusCode, _ := ResolveProjectRoute(req, "example.com", "path", mockGetBp, mockListSvcs)

		if statusCode != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", statusCode)
		}
	})
}
