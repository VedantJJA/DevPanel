package docker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/VedantJJA/devpnl/internal/db"
)

// HandleListBlueprints returns an HTTP handler for GET /api/blueprints.
func HandleListBlueprints(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if database == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"blueprints": []db.BlueprintRecord{}})
			return
		}

		blueprints, err := database.ListBlueprints(r.Context())
		if err != nil {
			log.Printf("api: list blueprints error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("Failed to list blueprints: %v", err)})
			return
		}

		if blueprints == nil {
			blueprints = []db.BlueprintRecord{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"blueprints": blueprints,
		})
	}
}

// HandleDeleteBlueprint returns an HTTP handler for DELETE /api/blueprints/delete?id=...
func HandleDeleteBlueprint(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
			return
		}

		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Blueprint ID parameter is required"})
			return
		}

		if database != nil {
			if err := database.DeleteBlueprint(r.Context(), id); err != nil {
				log.Printf("api: delete blueprint %s error: %v", id, err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("Failed to delete blueprint: %v", err)})
				return
			}
		}

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Blueprint deleted successfully",
		})
	}
}

// HandleDeployBlueprint returns an HTTP handler for POST /api/blueprints/deploy
func HandleDeployBlueprint(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
			return
		}

		var req struct {
			RepoURL string `json:"repo_url"`
			AppName string `json:"app_name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request payload"})
			return
		}

		if req.RepoURL == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Repository URL is required"})
			return
		}

		orchestrator := NewBlueprintOrchestrator(dockClient)

		// Create default blueprint for deployment trigger
		bp := &Blueprint{
			Version: "1.0",
			Project: req.AppName,
			Services: map[string]ServiceConfig{
				"web": {
					Type: "web",
					Source: SourceConfig{
						Repo: req.RepoURL,
					},
					Build: BuildConfig{
						Engine: "dockerfile",
					},
				},
			},
		}

		res, err := orchestrator.DeployOrchestrate(r.Context(), bp)
		if err != nil {
			log.Printf("api: deploy blueprint error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("Deployment failed: %v", err)})
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}
