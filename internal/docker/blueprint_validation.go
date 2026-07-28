package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

		// 1. Create a secure temporary directory for cloning
		tempDir, err := os.MkdirTemp("", "devpanel-val-*")
		if err != nil {
			log.Printf("blueprint-val: failed to create temp dir: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to initialize indexing workspace"})
			return
		}

		// CRITICAL: Securely clean up the temporary directory regardless of outcome
		defer func() {
			if removeErr := os.RemoveAll(tempDir); removeErr != nil {
				log.Printf("blueprint-val: cleanup temp dir %s error: %v", tempDir, removeErr)
			}
		}()

		// 2. Perform a shallow clone (Depth 1) using go-git/v5
		log.Printf("blueprint-val: shallow cloning %s into %s", req.RepoURL, tempDir)
		_, err = git.PlainCloneContext(r.Context(), tempDir, false, &git.CloneOptions{
			URL:          req.RepoURL,
			Depth:        1,
			SingleBranch: true,
		})
		if err != nil {
			log.Printf("blueprint-val: git clone error for %s: %v", req.RepoURL, err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: fmt.Sprintf("Failed to clone repository: %v. Please verify the URL and public repository access.", err),
			})
			return
		}

		// 3. Check for devpanel.yaml in the root of the repository
		yamlPath := filepath.Join(tempDir, "devpanel.yaml")
		if _, err := os.Stat(yamlPath); errors.Is(err, os.ErrNotExist) {
			// Fallback check for devpanel.yml
			yamlPath = filepath.Join(tempDir, "devpanel.yml")
			if _, err := os.Stat(yamlPath); errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: "Blueprint not found. Please ensure a devpanel.yaml file exists in the root of your repository.",
				})
				return
			}
		}

		// 4. Read devpanel.yaml file content
		yamlBytes, err := os.ReadFile(yamlPath)
		if err != nil {
			log.Printf("blueprint-val: failed to read %s: %v", yamlPath, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read devpanel.yaml from repository"})
			return
		}

		// 5. Parse and validate YAML schema
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

		// Override or sync project name if not specified
		if bp.Project == "" {
			bp.Project = req.AppName
		}

		// 6. Save blueprint dynamically into SQLite DB if available
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
