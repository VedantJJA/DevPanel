package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/VedantJJA/devpnl/internal/db"
)

// ScanService is the per-service shape returned to the wizard UI.
type ScanService struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	Image   string        `json:"image,omitempty"`
	Source  SourceConfig  `json:"source"`
	Build   BuildConfig   `json:"build"`
	Deploy  DeployConfig  `json:"deploy"`
	Default struct {
		Env  map[string]string `json:"env"`
		Port int               `json:"port"`
	} `json:"defaults"`
}

// ScanResult is the response for POST /api/repos/scan.
type ScanResult struct {
	Project   string        `json:"project"`
	RepoURL   string        `json:"repo_url"`
	Services  []ScanService `json:"services"`
	Warnings  []string      `json:"warnings"`
	Errors    []string      `json:"errors"`
	Blueprint *Blueprint    `json:"blueprint,omitempty"`
}

// HandleScanRepo scans a repository's devpanel.yaml and returns a structured result
// WITHOUT deploying. Used by wizard step 1.
func HandleScanRepo(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
			return
		}
		var req ValidateBlueprintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON request payload"})
			return
		}
		req.AppName = strings.TrimSpace(req.AppName)
		req.RepoURL = strings.TrimSpace(req.RepoURL)
		if req.RepoURL == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "repo_url is required"})
			return
		}

		result := ScanResult{RepoURL: req.RepoURL, Project: req.AppName}

		// 1. Fetch devpanel.yaml (raw API first, shallow-clone fallback)
		yamlBytes, err := fetchRawBlueprintContent(r.Context(), database, req.RepoURL)
		if err != nil {
			yamlBytes, err = fetchBlueprintViaShallowClone(r.Context(), req.RepoURL)
			if err != nil {
				result.Errors = append(result.Errors,
					"No devpanel.yaml found in repository root. Add one to continue.")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(result)
				return
			}
			result.Warnings = append(result.Warnings,
				"devpanel.yaml fetched via shallow clone (raw API unavailable).")
		}

		// 2. Parse YAML
		var bp Blueprint
		if err := yamlUnmarshal(yamlBytes, &bp); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid YAML syntax: %v", err))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(result)
			return
		}

		// 3. Validate schema (collect all errors, not just first)
		result.Errors = append(result.Errors, validateBlueprintAll(&bp)...)

		if bp.Project == "" {
			bp.Project = req.AppName
		}
		result.Project = bp.Project
		result.Blueprint = &bp

		// 4. Shape services for the UI
		for name, cfg := range bp.Services {
			s := ScanService{
				Name: name, Type: cfg.Type, Image: cfg.Image,
				Source: cfg.Source, Build: cfg.Build, Deploy: cfg.Deploy,
			}
			s.Default.Env = cfg.Deploy.Env
			if s.Default.Env == nil {
				s.Default.Env = map[string]string{}
			}
			s.Default.Port = cfg.Deploy.Port
			if cfg.Type == "web" && s.Default.Port == 0 {
				s.Default.Port = 8080
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Service %q has no port; defaulting to 8080.", name))
			}
			if cfg.Type == "database" && cfg.Image == "" {
				result.Errors = append(result.Errors,
					fmt.Sprintf("Service %q: database type requires 'image'.", name))
			}
			result.Services = append(result.Services, s)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}

var yamlUnmarshal = func(data []byte, v interface{}) error {
	return yamlLibUnmarshal(data, v)
}

func validateBlueprintAll(b *Blueprint) []string {
	var errs []string
	if len(b.Services) == 0 {
		return []string{"Blueprint must define at least one service."}
	}
	for name, cfg := range b.Services {
		if strings.TrimSpace(name) == "" {
			errs = append(errs, "Service name cannot be empty.")
			continue
		}
		if cfg.Type == "" {
			errs = append(errs, fmt.Sprintf("Service %q: missing 'type'.", name))
			continue
		}
		switch cfg.Type {
		case "database":
			if strings.TrimSpace(cfg.Image) == "" {
				errs = append(errs, fmt.Sprintf("Service %q: database requires 'image' (e.g. postgres:15).", name))
			}
		case "web", "static", "worker":
			// valid types
		default:
			errs = append(errs, fmt.Sprintf("Service %q: unsupported type %q (use web|static|database|worker).", name, cfg.Type))
		}
	}
	return errs
}
