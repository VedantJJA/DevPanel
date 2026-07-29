package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VedantJJA/devpnl/internal/db"
	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v3"
)

// ValidateBlueprintRequest represents the incoming JSON request for validating a GitHub repository blueprint.
type ValidateBlueprintRequest struct {
	AppName string `json:"app_name"`
	RepoURL string `json:"repo_url"`
}

// ValidateBlueprintResponse represents the success response payload.
type ValidateBlueprintResponse struct {
	Message   string     `json:"message"`
	Blueprint *Blueprint `json:"blueprint"`
}

// ErrorResponse represents structured error feedback for the frontend.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HandleValidateBlueprint handles POST /api/blueprints/validate
func HandleValidateBlueprint(database *db.DB) http.HandlerFunc {
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

		if req.AppName == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "App Name is required"})
			return
		}

		if req.RepoURL == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "GitHub Repository URL is required"})
			return
		}

		log.Printf("blueprint-val: fetching raw devpanel.yaml for %s without cloning entire repository", req.RepoURL)

		// 1. Fetch ONLY the devpanel.yaml file content via raw HTTP APIs
		yamlBytes, err := fetchRawBlueprintContent(r.Context(), database, req.RepoURL)
		if err != nil {
			log.Printf("blueprint-val: raw fetch failed for %s: %v. Attempting lightweight sparse fallback...", req.RepoURL, err)

			// Fallback: Perform a lightweight shallow clone into a temporary directory if raw API is unavailable
			yamlBytes, err = fetchBlueprintViaShallowClone(r.Context(), req.RepoURL)
			if err != nil {
				log.Printf("blueprint-val: validation failed for %s: %v", req.RepoURL, err)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: "Blueprint not found. Please ensure a devpanel.yaml file exists in the root of your repository.",
				})
				return
			}
		}

		// 2. Parse and validate YAML schema
		var bp Blueprint
		if err := yaml.Unmarshal(yamlBytes, &bp); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: fmt.Sprintf("Invalid YAML syntax: %v", err),
			})
			return
		}

		if err := bp.Validate(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: fmt.Sprintf("Invalid Blueprint schema: %v", err),
			})
			return
		}

		if bp.Project == "" {
			bp.Project = req.AppName
		}

		// 3. Save blueprint record into SQLite DB if available
		if database != nil {
			rec := &db.BlueprintRecord{
				ID:           fmt.Sprintf("bp-%s", sanitizeName(req.AppName)),
				Name:         req.AppName,
				RepoURL:      req.RepoURL,
				Status:       "valid",
				ServiceCount: len(bp.Services),
			}
			if err := database.CreateBlueprint(r.Context(), rec); err != nil {
				log.Printf("blueprint-val: notice saving blueprint to db: %v", err)
			}
		}

		log.Printf("blueprint-val: repository %s successfully validated (project: %s)", req.RepoURL, bp.Project)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ValidateBlueprintResponse{
			Message:   "Blueprint validated successfully",
			Blueprint: &bp,
		})
	}
}

// fetchRawBlueprintContent fetches only devpanel.yaml via raw HTTP API without cloning the repository.
func fetchRawBlueprintContent(ctx context.Context, database *db.DB, repoURL string) ([]byte, error) {
	// Format GitHub raw URLs
	// e.g. https://github.com/username/repo -> https://raw.githubusercontent.com/username/repo/main/devpanel.yaml
	trimmed := strings.TrimSuffix(repoURL, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")

	var candidateURLs []string

	if strings.Contains(trimmed, "github.com/") {
		parts := strings.Split(trimmed, "github.com/")
		if len(parts) == 2 {
			repoPath := parts[1]
			candidateURLs = []string{
				fmt.Sprintf("https://raw.githubusercontent.com/%s/main/devpanel.yaml", repoPath),
				fmt.Sprintf("https://raw.githubusercontent.com/%s/main/devpanel.yml", repoPath),
				fmt.Sprintf("https://raw.githubusercontent.com/%s/master/devpanel.yaml", repoPath),
				fmt.Sprintf("https://raw.githubusercontent.com/%s/master/devpanel.yml", repoPath),
			}
		}
	}

	client := &http.Client{Timeout: 8 * time.Second}

	var ghToken string
	if database != nil {
		if tok, err := database.GetSetting(ctx, "github_token"); err == nil && tok != "" {
			ghToken = tok
		}
	}

	for _, urlStr := range candidateURLs {
		req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if err != nil {
			continue
		}
		if ghToken != "" {
			req.Header.Set("Authorization", "Bearer "+ghToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			data, err := io.ReadAll(resp.Body)
			if err == nil && len(data) > 0 {
				log.Printf("blueprint-val: successfully fetched raw file from %s (%d bytes)", urlStr, len(data))
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("devpanel.yaml not found via raw HTTP API for %s", repoURL)
}

// fetchBlueprintViaShallowClone acts as a fallback for non-GitHub or private repositories.
func fetchBlueprintViaShallowClone(ctx context.Context, repoURL string) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "devpanel-val-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	_, err = git.PlainCloneContext(ctx, tempDir, false, &git.CloneOptions{
		URL:          repoURL,
		Depth:        1,
		SingleBranch: true,
	})
	if err != nil {
		return nil, err
	}

	yamlPath := filepath.Join(tempDir, "devpanel.yaml")
	if data, err := os.ReadFile(yamlPath); err == nil {
		return data, nil
	}

	yamlPath = filepath.Join(tempDir, "devpanel.yml")
	if data, err := os.ReadFile(yamlPath); err == nil {
		return data, nil
	}

	return nil, fmt.Errorf("devpanel.yaml not found in repository root")
}
