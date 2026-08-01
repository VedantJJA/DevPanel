// Package caddy provides a client for Caddy's admin API to dynamically
// manage reverse-proxy routes and On-Demand TLS.
//
// It communicates with Caddy's admin endpoint (default http://localhost:2019)
// using PATCH requests to add/remove routes at runtime without reloading.
package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client talks to Caddy's admin API.
type Client struct {
	adminURL string
	http     *http.Client
}

// NewClient creates a Caddy admin API client.
// Pass "" to use the default http://localhost:2019.
func NewClient(adminURL string) *Client {
	if adminURL == "" {
		adminURL = "http://localhost:2019"
	}
	return &Client{
		adminURL: adminURL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ---------- Route Management ------------------------------------------------

// Route represents a simplified Caddy HTTP route for reverse proxying.
type Route struct {
	ID       string `json:"@id"`                 // unique identifier for PATCH/DELETE
	Match    []Match `json:"match"`
	Handle   []Handler `json:"handle"`
	Terminal bool   `json:"terminal"`
}

// Match defines which requests a route handles.
type Match struct {
	Host []string `json:"host,omitempty"`
}

// Handler defines what to do with matched requests.
type Handler struct {
	Handler   string     `json:"handler"`
	Upstreams []Upstream `json:"upstreams,omitempty"`
}

// Upstream is a backend target for reverse_proxy.
type Upstream struct {
	Dial string `json:"dial"` // e.g. "localhost:8080"
}

// AddRoute dynamically adds a reverse-proxy route to Caddy mapping
// the given FQDN to a backend address (host:port).
func (c *Client) AddRoute(ctx context.Context, routeID, fqdn, backendAddr string) error {
	route := Route{
		ID: routeID,
		Match: []Match{
			{Host: []string{fqdn}},
		},
		Handle: []Handler{
			{
				Handler: "reverse_proxy",
				Upstreams: []Upstream{
					{Dial: backendAddr},
				},
			},
		},
		Terminal: true,
	}

	body, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("caddy: marshal route: %w", err)
	}

	// PATCH to append the route to the HTTP server's routes array.
	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", c.adminURL)
	log.Printf("caddy: adding route %s → %s (%s)", fqdn, backendAddr, routeID)

	return c.patchAppend(ctx, url, body)
}

// RemoveRoute removes a route by its @id.
func (c *Client) RemoveRoute(ctx context.Context, routeID string) error {
	url := fmt.Sprintf("%s/id/%s", c.adminURL, routeID)
	log.Printf("caddy: removing route %s", routeID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("caddy: delete request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy: delete route %s: %w", routeID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy: delete route %s: HTTP %d: %s", routeID, resp.StatusCode, body)
	}

	return nil
}

// ---------- On-Demand TLS Config --------------------------------------------

// ConfigureOnDemandTLS sets up Caddy's On-Demand TLS to verify domains
// by calling the given askURL endpoint (e.g. "http://localhost:8090/ask").
func (c *Client) ConfigureOnDemandTLS(ctx context.Context, askURL string) error {
	config := map[string]interface{}{
		"on_demand": map[string]interface{}{
			"permission": map[string]interface{}{
				"endpoint": askURL,
				"module":   "http",
			},
		},
	}

	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("caddy: marshal tls config: %w", err)
	}

	url := fmt.Sprintf("%s/config/apps/tls/automation/policies/0/issuers/0/challenges/http", c.adminURL)
	log.Printf("caddy: configuring On-Demand TLS → %s", askURL)

	return c.patch(ctx, url, body)
}

// GenerateCaddyfileBlock generates a strict Caddyfile reverse proxy block:
// <project-name>.<domain> {
//     reverse_proxy <containerTarget>
// }
func GenerateCaddyfileBlock(projectName, domain, containerTarget string) string {
	cleanDomain := strings.TrimPrefix(domain, "panel.")
	fqdn := fmt.Sprintf("%s.%s", projectName, cleanDomain)
	return fmt.Sprintf("%s {\n\treverse_proxy %s\n}\n", fqdn, containerTarget)
}

// LoadCaddyfile sends a Caddyfile payload to Caddy's admin endpoint (POST /load) to apply config instantly.
func (c *Client) LoadCaddyfile(ctx context.Context, caddyfileContent string) error {
	url := fmt.Sprintf("%s/load", c.adminURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(caddyfileContent)))
	if err != nil {
		return fmt.Errorf("caddy: create load request: %w", err)
	}
	req.Header.Set("Content-Type", "text/caddyfile")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy: load caddyfile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy: load caddyfile: HTTP %d: %s", resp.StatusCode, body)
	}
	log.Printf("caddy: successfully loaded Caddyfile configuration via POST /load")
	return nil
}

// ---------- Health ----------------------------------------------------------

// Healthy checks if the Caddy admin API is reachable.
func (c *Client) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.adminURL+"/config/", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ---------- Internal helpers ------------------------------------------------

// patchAppend sends a PATCH request that appends an element to a JSON array.
func (c *Client) patchAppend(ctx context.Context, url string, body []byte) error {
	// Caddy PATCH on an array path with a JSON object appends it.
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("caddy: patch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy: patch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy: patch %s: HTTP %d: %s", url, resp.StatusCode, respBody)
	}

	return nil
}

// patch sends a PATCH request to set a value at the given config path.
func (c *Client) patch(ctx context.Context, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("caddy: patch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy: patch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy: patch %s: HTTP %d: %s", url, resp.StatusCode, respBody)
	}

	return nil
}

// RouteID generates a deterministic route ID for a project+domain pair.
func RouteID(projectName, fqdn string) string {
	return fmt.Sprintf("devpnl-%s-%s", projectName, fqdn)
}
