package caddy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddRoute_SendsCorrectPayload(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedMethod, receivedPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	err := c.AddRoute(context.Background(), "devpnl-myapp-app.example.com", "app.example.com", "localhost:3000")
	if err != nil {
		t.Fatalf("AddRoute: %v", err)
	}

	if receivedMethod != "PATCH" {
		t.Errorf("expected PATCH, got %s", receivedMethod)
	}

	if receivedPath != "/config/apps/http/servers/srv0/routes" {
		t.Errorf("unexpected path: %s", receivedPath)
	}

	if id, ok := receivedBody["@id"].(string); !ok || id != "devpnl-myapp-app.example.com" {
		t.Errorf("unexpected @id: %v", receivedBody["@id"])
	}
}

func TestRemoveRoute_SendsDelete(t *testing.T) {
	var receivedMethod, receivedPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	err := c.RemoveRoute(context.Background(), "devpnl-myapp-app.example.com")
	if err != nil {
		t.Fatalf("RemoveRoute: %v", err)
	}

	if receivedMethod != "DELETE" {
		t.Errorf("expected DELETE, got %s", receivedMethod)
	}

	if receivedPath != "/id/devpnl-myapp-app.example.com" {
		t.Errorf("unexpected path: %s", receivedPath)
	}
}

func TestHealthy_ReturnsTrue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	if !c.Healthy(context.Background()) {
		t.Error("expected Healthy=true")
	}
}

func TestHealthy_ReturnsFalseOnError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1") // nothing listens here
	if c.Healthy(context.Background()) {
		t.Error("expected Healthy=false for unreachable server")
	}
}

func TestRouteID(t *testing.T) {
	id := RouteID("myapp", "app.example.com")
	if id != "devpnl-myapp-app.example.com" {
		t.Errorf("unexpected route ID: %s", id)
	}
}
