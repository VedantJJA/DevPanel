package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/VedantJJA/devpnl/internal/db"
)

type createProjectRequest struct {
	AppName   string             `json:"app_name"`
	RepoURL   string             `json:"repo_url"`
	Blueprint *Blueprint         `json:"blueprint"`
	Services  []createServiceIn  `json:"services"`
}

type createServiceIn struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	EnvVars      map[string]string `json:"env_vars"`
	Port         int               `json:"port"`
	CustomDomain string            `json:"custom_domain"`
	AutoDeploy   bool              `json:"auto_deploy"`
	BuildCommand string            `json:"build_command"`
	StartCommand string            `json:"start_command"`
	InstanceType string            `json:"instance_type"`
}

type projectOut struct {
	Blueprint db.BlueprintRecord   `json:"blueprint"`
	Services  []db.ServiceRecord   `json:"services"`
	Latest    *db.DeploymentRecord `json:"latest,omitempty"`
}

type updateServiceRequest struct {
	EnvVars      *map[string]string `json:"env_vars,omitempty"`
	Port         *int               `json:"port,omitempty"`
	CustomDomain *string            `json:"custom_domain,omitempty"`
	AutoDeploy   *bool              `json:"auto_deploy,omitempty"`
	BuildCommand *string            `json:"build_command,omitempty"`
	StartCommand *string            `json:"start_command,omitempty"`
	InstanceType *string            `json:"instance_type,omitempty"`
}

func newDeploymentID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "dep_" + hex.EncodeToString(b)
}

func orDefault(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}

// HandleCreateProject — POST /api/projects
func HandleCreateProject(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
			return
		}
		var req createProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON"})
			return
		}
		if req.AppName == "" || req.RepoURL == "" || req.Blueprint == nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "app_name, repo_url and blueprint are required"})
			return
		}
		projectID := fmt.Sprintf("bp-%s", sanitizeName(req.AppName))

		// Persist blueprint record
		if err := database.CreateBlueprint(r.Context(), &db.BlueprintRecord{
			ID: projectID, Name: req.AppName, RepoURL: req.RepoURL,
			Status: "ready", ServiceCount: len(req.Services),
		}); err != nil {
			log.Printf("api: create project blueprint: %v", err)
		}

		// Persist per-service settings
		for _, s := range req.Services {
			rec := &db.ServiceRecord{
				ProjectID: projectID, Name: s.Name, Type: s.Type,
				EnvVars: s.EnvVars, Port: s.Port, CustomDomain: s.CustomDomain,
				AutoDeploy: s.AutoDeploy, BuildCommand: s.BuildCommand,
				StartCommand: s.StartCommand, InstanceType: orDefault(s.InstanceType, "free"),
			}
			if err := database.UpsertService(r.Context(), rec); err != nil {
				log.Printf("api: upsert service %s: %v", s.Name, err)
			}
		}

		svcs, _ := database.ListServices(r.Context(), projectID)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(projectOut{
			Blueprint: db.BlueprintRecord{
				ID: projectID, Name: req.AppName, RepoURL: req.RepoURL,
				Status: "ready", ServiceCount: len(svcs),
			},
			Services: svcs,
		})
	}
}

// HandleListProjects — GET /api/projects
func HandleListProjects(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bps, err := database.ListBlueprints(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}
		type projectListItem struct {
			db.BlueprintRecord
			Services int `json:"service_count_actual"`
		}
		out := make([]projectListItem, 0, len(bps))
		for _, b := range bps {
			svcs, _ := database.ListServices(r.Context(), b.ID)
			out = append(out, projectListItem{BlueprintRecord: b, Services: len(svcs)})
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"projects": out})
	}
}

// HandleGetProject — GET /api/projects/{id}
func HandleGetProject(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := r.PathValue("id")
		if id == "" {
			id = r.URL.Query().Get("id")
		}
		bps, err := database.ListBlueprints(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}
		var found db.BlueprintRecord
		match := false
		for _, b := range bps {
			if b.ID == id {
				found = b
				match = true
				break
			}
		}
		if !match {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "project not found"})
			return
		}
		svcs, _ := database.ListServices(r.Context(), id)
		deps, _ := database.ListDeployments(r.Context(), id)
		var latest *db.DeploymentRecord
		if len(deps) > 0 {
			latest = &deps[0]
		}
		json.NewEncoder(w).Encode(projectOut{Blueprint: found, Services: svcs, Latest: latest})
	}
}

// HandleUpdateService — PATCH /api/projects/{id}/services/{name}
func HandleUpdateService(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPatch && r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Use PATCH"})
			return
		}
		projectID := r.PathValue("id")
		name := r.PathValue("name")
		existing, err := database.GetService(r.Context(), projectID, name)
		if err != nil || existing == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "service not found"})
			return
		}
		var req updateServiceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON"})
			return
		}
		if req.EnvVars != nil {
			existing.EnvVars = *req.EnvVars
		}
		if req.Port != nil {
			existing.Port = *req.Port
		}
		if req.CustomDomain != nil {
			existing.CustomDomain = *req.CustomDomain
		}
		if req.AutoDeploy != nil {
			existing.AutoDeploy = *req.AutoDeploy
		}
		if req.BuildCommand != nil {
			existing.BuildCommand = *req.BuildCommand
		}
		if req.StartCommand != nil {
			existing.StartCommand = *req.StartCommand
		}
		if req.InstanceType != nil {
			existing.InstanceType = *req.InstanceType
		}
		if err := database.UpsertService(r.Context(), existing); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(existing)
	}
}

// HandleTriggerProjectDeploy — POST /api/projects/{id}/deploy
func HandleTriggerProjectDeploy(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
			return
		}
		projectID := r.PathValue("id")

		bps, _ := database.ListBlueprints(r.Context())
		var bpRec db.BlueprintRecord
		found := false
		for _, b := range bps {
			if b.ID == projectID {
				bpRec = b
				found = true
				break
			}
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "project not found"})
			return
		}
		svcs, err := database.ListServices(r.Context(), projectID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}

		yamlBytes, err := fetchRawBlueprintContent(r.Context(), bpRec.RepoURL)
		if err != nil {
			yamlBytes, err = fetchBlueprintViaShallowClone(r.Context(), bpRec.RepoURL)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Could not fetch devpanel.yaml from repo"})
				return
			}
		}
		var bp Blueprint
		if err := yamlUnmarshal(yamlBytes, &bp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid blueprint YAML in repo"})
			return
		}
		bp.Project = bpRec.Name
		overrides := map[string]db.ServiceRecord{}
		for _, s := range svcs {
			overrides[s.Name] = s
		}
		for name, cfg := range bp.Services {
			ov, ok := overrides[name]
			if !ok {
				continue
			}
			if cfg.Deploy.Env == nil {
				cfg.Deploy.Env = map[string]string{}
			}
			for k, v := range ov.EnvVars {
				cfg.Deploy.Env[k] = v
			}
			if ov.Port > 0 {
				cfg.Deploy.Port = ov.Port
			}
			if ov.StartCommand != "" {
				cfg.Deploy.Command = ov.StartCommand
			}
			bp.Services[name] = cfg
		}

		depID := newDeploymentID()
		_ = database.CreateDeployment(r.Context(), &db.DeploymentRecord{
			ID: depID, ProjectID: projectID, Status: "building", Trigger: "manual",
		})
		_ = database.CreateBlueprint(r.Context(), &db.BlueprintRecord{
			ID: projectID, Name: bpRec.Name, RepoURL: bpRec.RepoURL,
			Status: "deploying", ServiceCount: len(bp.Services),
		})

		go func() {
			bg := context.Background()
			orch := NewBlueprintOrchestrator(dockClient)
			orch.ProjectID = projectID
			orch.Sink = globalLogBroadcaster.Broadcast

			globalLogBroadcaster.Broadcast(projectID, "system", "engine",
				fmt.Sprintf("Starting deployment %s for %q (%d services)", depID, bpRec.Name, len(bp.Services)), "info")
			globalLogBroadcaster.Broadcast(projectID, "clone", "git",
				fmt.Sprintf("Cloning %s ...", bpRec.RepoURL), "info")

			res, err := orch.DeployOrchestrate(bg, &bp, bpRec.RepoURL)
			if err != nil {
				globalLogBroadcaster.Broadcast(projectID, "deploy", "engine",
					fmt.Sprintf("Deployment failed: %v", err), "error")
				_ = database.UpdateDeploymentStatus(bg, depID, "error", err.Error())
				_ = database.CreateBlueprint(bg, &db.BlueprintRecord{
					ID: projectID, Name: bpRec.Name, RepoURL: bpRec.RepoURL, Status: "error",
				})
				return
			}
			globalLogBroadcaster.Broadcast(projectID, "runtime", "engine",
				fmt.Sprintf("Live — %d services running.", len(res.Services)), "success")
			_ = database.UpdateDeploymentStatus(bg, depID, "live", "")
			_ = database.CreateBlueprint(bg, &db.BlueprintRecord{
				ID: projectID, Name: bpRec.Name, RepoURL: bpRec.RepoURL,
				Status: "active", ServiceCount: len(res.Services),
			})
		}()

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"deployment_id": depID,
			"project_id":    projectID,
			"status":        "building",
			"log_url":       fmt.Sprintf("/api/projects/%s/logs", projectID),
		})
	}
}

// HandleListDeployments — GET /api/projects/{id}/deployments
func HandleListDeployments(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := r.PathValue("id")
		deps, err := database.ListDeployments(r.Context(), id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}
		if deps == nil {
			deps = []db.DeploymentRecord{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"deployments": deps})
	}
}

// HandleProjectLogsSSE — GET /api/projects/{id}/logs?service=<name>
func HandleProjectLogsSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		projectID := r.PathValue("id")
		if projectID == "" {
			http.Error(w, "project id required", http.StatusBadRequest)
			return
		}
		serviceFilter := strings.TrimSpace(r.URL.Query().Get("service"))

		ch, unsubscribe := globalLogBroadcaster.Subscribe(projectID)
		defer unsubscribe()
		fmt.Fprintf(w, "event: connected\ndata: {\"project\":\"%s\"}\n\n", projectID)
		flusher.Flush()
		ctx := r.Context()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case evt, ok := <-ch:
				if !ok {
					return
				}
				if serviceFilter != "" && evt.Service != serviceFilter && evt.Service != "engine" && evt.Service != "git" {
					continue
				}
				data, err := json.Marshal(evt)
				if err == nil {
					fmt.Fprintf(w, "data: %s\n\n", string(data))
					flusher.Flush()
				}
			}
		}
	}
}

// HandleRestartService — POST /api/projects/{id}/services/{name}/restart
func HandleRestartService(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		projectID := r.PathValue("id")
		name := r.PathValue("name")
		containerName := fmt.Sprintf("devpnl-%s-%s", sanitizeName(projectID), sanitizeName(name))
		containers, err := dockClient.ListContainers(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}
		var cid string
		for _, c := range containers {
			if len(c.Names) > 0 && strings.TrimPrefix(c.Names[0], "/") == containerName {
				cid = c.ID
				break
			}
		}
		if cid == "" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "running container not found"})
			return
		}
		if err := dockClient.StopContainer(r.Context(), cid); err != nil {
			log.Printf("api: restart stop container: %v", err)
		}
		if err := dockClient.StartContainer(r.Context(), cid); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("start container failed: %v", err)})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "restarted", "container_id": cid})
	}
}
