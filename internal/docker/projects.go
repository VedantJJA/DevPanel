package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VedantJJA/devpnl/internal/db"
)

type createProjectRequest struct {
	AppName   string            `json:"app_name"`
	RepoURL   string            `json:"repo_url"`
	Blueprint interface{}       `json:"blueprint"`
	Services  []createServiceIn `json:"services"`
}

type createServiceIn struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Image        string            `json:"image"`
	EnvVars      map[string]string `json:"env_vars"`
	Port         int               `json:"port"`
	CustomDomain string            `json:"custom_domain"`
	AutoDeploy   bool              `json:"auto_deploy"`
	BuildCommand string            `json:"build_command"`
	StartCommand string            `json:"start_command"`
	InstanceType string            `json:"instance_type"`
	Runtime      string            `json:"runtime"`
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
	Runtime      *string            `json:"runtime,omitempty"`
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

// generateUniqueSlug creates a Render-style unique URL slug for a service.
// It tries the sanitized baseName first; if already taken, appends -xxxx
// (4 random alphanumeric characters) and retries up to 5 times.
func generateUniqueSlug(ctx context.Context, database *db.DB, baseName string) string {
	slug := sanitizeName(baseName)
	if slug == "" {
		slug = "svc"
	}

	exists, err := database.SlugExists(ctx, slug)
	if err == nil && !exists {
		return slug
	}

	// Slug is taken — append random suffix
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	for attempt := 0; attempt < 5; attempt++ {
		b := make([]byte, 4)
		_, _ = rand.Read(b)
		suffix := make([]byte, 4)
		for i := range b {
			suffix[i] = charset[int(b[i])%len(charset)]
		}
		candidate := fmt.Sprintf("%s-%s", slug, string(suffix))
		exists, err := database.SlugExists(ctx, candidate)
		if err == nil && !exists {
			return candidate
		}
	}

	// Extremely unlikely fallback: use full hex
	fb := make([]byte, 4)
	_, _ = rand.Read(fb)
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(fb))
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
			log.Printf("api: decode createProjectRequest error: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("Invalid JSON request body: %v", err)})
			return
		}

		if req.AppName == "" {
			if bpMap, ok := req.Blueprint.(map[string]interface{}); ok {
				if n, ok := bpMap["name"].(string); ok && n != "" {
					req.AppName = n
				} else if p, ok := bpMap["project"].(string); ok && p != "" {
					req.AppName = p
				}
			}
		}
		if req.AppName == "" && req.RepoURL != "" {
			parts := strings.Split(strings.TrimSuffix(req.RepoURL, ".git"), "/")
			if len(parts) > 0 {
				req.AppName = parts[len(parts)-1]
			}
		}
		if req.AppName == "" {
			req.AppName = "my-app"
		}

		projectID := fmt.Sprintf("bp-%s", sanitizeName(req.AppName))

		// Persist blueprint record
		if err := database.CreateBlueprint(r.Context(), &db.BlueprintRecord{
			ID: projectID, Name: req.AppName, RepoURL: req.RepoURL,
			Status: "ready", ServiceCount: len(req.Services),
		}); err != nil {
			log.Printf("api: create project blueprint: %v", err)
		}

		// Persist per-service settings with Render-style unique URL slugs
		for _, s := range req.Services {
			slug := generateUniqueSlug(r.Context(), database, s.Name)
			rec := &db.ServiceRecord{
				ProjectID: projectID, Name: s.Name, Slug: slug, Type: s.Type, Image: s.Image,
				EnvVars: s.EnvVars, Port: s.Port, CustomDomain: s.CustomDomain,
				AutoDeploy: s.AutoDeploy, BuildCommand: s.BuildCommand,
				StartCommand: s.StartCommand, InstanceType: orDefault(s.InstanceType, "free"),
				Runtime: s.Runtime,
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
func HandleGetProject(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := r.PathValue("id")
		if id == "" {
			id = r.URL.Query().Get("id")
		}
		bp, err := database.GetBlueprint(r.Context(), id)
		if err != nil || bp == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("project not found: %s", id)})
			return
		}

		svcs, _ := database.ListServices(r.Context(), bp.ID)
		if dockClient != nil {
			containers, _ := dockClient.ListContainers(r.Context())
			for i := range svcs {
				livePort := findContainerPort(containers, bp.ID, bp.Name, svcs[i].Name, svcs[i].Port)
				if livePort > 0 {
					svcs[i].Port = livePort
				}
			}
		}

		deps, _ := database.ListDeployments(r.Context(), bp.ID)
		var latest *db.DeploymentRecord
		if len(deps) > 0 {
			latest = &deps[0]
		}
		json.NewEncoder(w).Encode(projectOut{Blueprint: *bp, Services: svcs, Latest: latest})
	}
}

// HandleDeleteProject — DELETE /api/projects/{id}
// Stops & removes all containers, purges build images & cache, deletes DB records, and clears logs.
func HandleDeleteProject(database *db.DB, dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Use DELETE"})
			return
		}
		projectID := r.PathValue("id")
		if projectID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "project id required"})
			return
		}

		projectSlug := strings.ToLower(strings.TrimPrefix(projectID, "bp-"))

		// 1. Stop & Remove known service containers from DB records
		svcs, _ := database.ListServices(r.Context(), projectID)
		for _, svc := range svcs {
			svcName := strings.ToLower(svc.Name)
			cName1 := fmt.Sprintf("devpnl-%s-%s", projectSlug, svcName)
			cName2 := fmt.Sprintf("devpnl-%s-%s", projectID, svcName)
			_ = dockClient.StopContainer(r.Context(), cName1)
			_ = dockClient.RemoveContainer(r.Context(), cName1, true)
			_ = dockClient.StopContainer(r.Context(), cName2)
			_ = dockClient.RemoveContainer(r.Context(), cName2, true)
		}

		// 2. Query Docker API to find and stop ALL containers matching project slug/id
		allContainers, err := dockClient.ListContainers(r.Context())
		if err == nil {
			for _, c := range allContainers {
				var nameClean string
				if len(c.Names) > 0 {
					nameClean = strings.TrimPrefix(c.Names[0], "/")
				} else {
					nameClean = strings.TrimPrefix(c.ID, "/")
				}
				if strings.HasPrefix(nameClean, fmt.Sprintf("devpnl-%s-", projectSlug)) ||
					strings.HasPrefix(nameClean, fmt.Sprintf("devpnl-%s-", projectID)) ||
					(projectSlug != "" && strings.Contains(nameClean, projectSlug)) {
					log.Printf("projects: stopping & removing container %s for deleted project %s", nameClean, projectID)
					_ = dockClient.StopContainer(r.Context(), nameClean)
					_ = dockClient.RemoveContainer(r.Context(), nameClean, true)
				}
			}
		}

		// 3. Purge temp build cache directories for this project
		tempDir := os.TempDir()
		if entries, err := os.ReadDir(tempDir); err == nil {
			prefix := fmt.Sprintf("devpanel-blueprint-%s-", projectSlug)
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
					targetPath := filepath.Join(tempDir, entry.Name())
					_ = os.RemoveAll(targetPath)
					log.Printf("projects: purged build cache directory %s", targetPath)
				}
			}
		}

		// 4. Re-sync Nginx dynamic routing state
		if err := globalNginxManager.SyncNginx(dockClient); err != nil {
			log.Printf("nginx_manager: sync failed after deleting project %s: %v", projectID, err)
		}

		// 5. Purge build logs from memory and disk
		globalLogBroadcaster.ClearLogs(projectID)

		// 6. Delete DB records (services, deployments, blueprint)
		if err := database.DeleteBlueprint(r.Context(), projectID); err != nil {
			log.Printf("api: delete project %s: %v", projectID, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"deleted": projectID})
	}
}

// HandleClearProjectLogs — DELETE /api/projects/{id}/logs
// Purges only the persisted log history without affecting containers or DB.
func HandleClearProjectLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Use DELETE"})
			return
		}
		projectID := r.PathValue("id")
		if projectID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "project id required"})
			return
		}
		globalLogBroadcaster.ClearLogs(projectID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"cleared": true})
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
		if req.Runtime != nil {
			existing.Runtime = *req.Runtime
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

		bpRec, err := database.GetBlueprint(r.Context(), projectID)
		if err != nil || bpRec == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("project not found: %s", projectID)})
			return
		}
		svcs, err := database.ListServices(r.Context(), bpRec.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}

		var bp Blueprint

		var yamlBytes []byte
		var fetchErr error
		if bpRec.RepoURL != "" {
			yamlBytes, fetchErr = fetchRawBlueprintContent(r.Context(), database, bpRec.RepoURL)
			if fetchErr != nil {
				yamlBytes, fetchErr = fetchBlueprintViaShallowClone(r.Context(), bpRec.RepoURL)
			}
		} else {
			fetchErr = fmt.Errorf("no repo_url")
		}

		if fetchErr == nil {
			// devpanel.yaml found
			if err := yamlUnmarshal(yamlBytes, &bp); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid blueprint YAML in repo"})
				return
			}
		} else {
			// Synthesize blueprint from DB services
			bp.Version = "1.0"
			bp.Services = make(map[string]ServiceConfig)
			for _, s := range svcs {
				sc := ServiceConfig{
					Type: s.Type,
					Image: s.Image,
					Deploy: DeployConfig{
						Port:    s.Port,
						Env:     s.EnvVars,
						Command: s.StartCommand,
					},
					Build: BuildConfig{
						Command: s.BuildCommand,
					},
				}
				if s.Image != "" {
					// Prebuilt image, no build engine needed
				} else if s.Type == "static" {
					sc.Build.Engine = "static"
					sc.Build.OutputDir = "dist" // Default
					sc.Source = SourceConfig{Directory: ".", Ref: "main"}
				} else if s.Type == "web" {
					sc.Build.Engine = "node"
					sc.Source = SourceConfig{Directory: ".", Ref: "main"}
				} else if s.Type == "cron" {
					sc.Build.Engine = "python"
					sc.Source = SourceConfig{Directory: ".", Ref: "main"}
				}
				bp.Services[s.Name] = sc
			}
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

func isLogEventMatch(evt LogEvent, filter string) bool {
	if filter == "" || filter == "all" {
		return true
	}
	s := strings.ToLower(evt.Service)
	f := strings.ToLower(filter)
	return s == f
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

		globalLogBroadcaster.loadHistory(projectID)
		globalLogBroadcaster.mu.RLock()
		pastLogs := append([]LogEvent(nil), globalLogBroadcaster.history[projectID]...)
		globalLogBroadcaster.mu.RUnlock()

		fmt.Fprintf(w, "event: connected\ndata: {\"project\":\"%s\"}\n\n", projectID)
		for _, evt := range pastLogs {
			if serviceFilter != "" && !isLogEventMatch(evt, serviceFilter) {
				continue
			}
			if data, err := json.Marshal(evt); err == nil {
				fmt.Fprintf(w, "data: %s\n\n", string(data))
			}
		}
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
				if serviceFilter != "" && !isLogEventMatch(evt, serviceFilter) {
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
		var cid string
		if err == nil {
			for _, c := range containers {
				for _, n := range c.Names {
					cleanName := strings.TrimPrefix(n, "/")
					if cleanName == containerName || (strings.Contains(cleanName, sanitizeName(projectID)) && strings.Contains(cleanName, sanitizeName(name))) {
						cid = c.ID
						break
					}
				}
				if cid != "" {
					break
				}
			}
		}
		if cid != "" {
			_ = dockClient.StopContainer(r.Context(), cid)
			_ = dockClient.StartContainer(r.Context(), cid)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":       "restarted",
			"message":      fmt.Sprintf("Service container %s restarted successfully.", name),
			"container_id": cid,
		})
	}
}
