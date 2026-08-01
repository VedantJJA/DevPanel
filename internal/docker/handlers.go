package docker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/VedantJJA/devpnl/internal/db"
)

// ContainerDTO represents the API response format for a container.
type ContainerDTO struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Image      string  `json:"image"`
	Status     string  `json:"status"`
	Port       string  `json:"port"`
	CPUPercent float64 `json:"cpuPercent"`
	MemoryMB   float64 `json:"memoryMb"`
	Uptime     string  `json:"uptime"`
}

// HandleListContainers returns an HTTP handler that lists all live containers from Docker.
func HandleListContainers(dockClient *Client, database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		containers, err := dockClient.ListContainers(r.Context())
		if err != nil {
			log.Printf("api: list containers error: %v", err)
			// Return empty list if Docker is unavailable (e.g. non-docker env)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"containers": []ContainerDTO{},
				"error":      err.Error(),
			})
			return
		}

		dtoList := make([]ContainerDTO, len(containers))
		var wg sync.WaitGroup

		for i, c := range containers {
			wg.Add(1)
			go func(idx int, c ContainerSummary) {
				defer wg.Done()
				name := "unnamed"
				if len(c.Names) > 0 {
					name = c.Names[0]
					if len(name) > 0 && name[0] == '/' {
						name = name[1:]
					}
				}

				status := "stopped"
				if c.State == "running" {
					status = "running"
				} else if c.State == "restarting" {
					status = "restarting"
				}

				shortID := c.ID
				if len(shortID) > 12 {
					shortID = shortID[:12]
				}

				var cpu, mem float64
				if status == "running" {
					// dockClient.Stats is concurrent-safe (it just does an HTTP request to Docker)
					stats, err := dockClient.Stats(r.Context(), c.ID)
					if err == nil && stats != nil {
						cpu = stats.CPUPercent
						mem = stats.MemUsageMB
					}
				}

				dtoList[idx] = ContainerDTO{
					ID:         shortID,
					Name:       name,
					Image:      c.Image,
					Status:     status,
					Port:       FormatPorts(c.Ports),
					CPUPercent: MathRound(cpu, 1),
					MemoryMB:   MathRound(mem, 1),
					Uptime:     c.Status,
				}
			}(i, c)
		}
		wg.Wait()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"containers": dtoList,
		})
	}
}

// HandleSystemStats returns live host & Docker telemetry automatically.
func HandleSystemStats(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		hostStats := GetHostMetrics()

		info, err := dockClient.SystemInfo(r.Context())
		if err != nil {
			log.Printf("api: system info notice: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"totalContainers":   0,
				"activeContainers":  0,
				"stoppedContainers": 0,
				"totalMemMb":        hostStats.TotalMemMB,
				"usedMemMb":         hostStats.UsedMemMB,
				"memPercent":        hostStats.MemPercent,
				"cpuPercent":        hostStats.CpuPercent,
				"cpus":              hostStats.CPUs,
				"os":                hostStats.OS,
				"arch":              hostStats.Arch,
				"notice":            err.Error(),
			})
			return
		}

		osName := info.OperatingSystem
		if osName == "" {
			osName = hostStats.OS
		}

		archName := info.Architecture
		if archName == "" {
			archName = hostStats.Arch
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"totalContainers":   info.Containers,
			"activeContainers":  info.ContainersRunning,
			"stoppedContainers": info.ContainersStopped,
			"totalMemMb":        hostStats.TotalMemMB,
			"usedMemMb":         hostStats.UsedMemMB,
			"memPercent":        hostStats.MemPercent,
			"cpuPercent":        hostStats.CpuPercent,
			"cpus":              hostStats.CPUs,
			"os":                osName,
			"arch":              archName,
		})
	}
}

// HandleListVolumes returns an HTTP handler that lists all live Docker volumes.
func HandleListVolumes(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		volumes, err := dockClient.ListVolumes(r.Context())
		if err != nil {
			log.Printf("api: list volumes notice: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"volumes": []VolumeSummary{},
				"notice":  err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"volumes": volumes,
		})
	}
}

// HandleStartContainer starts a specific container.
func HandleStartContainer(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			id = r.URL.Query().Get("id")
		}

		if err := dockClient.StartContainer(r.Context(), id); err != nil {
			log.Printf("api: start container %s error: %v", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}
}

// HandleStopContainer stops a specific container.
func HandleStopContainer(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			id = r.URL.Query().Get("id")
		}

		if err := dockClient.StopContainer(r.Context(), id); err != nil {
			log.Printf("api: stop container %s error: %v", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}
}

// HandleStartAllContainers starts all registered containers.
func HandleStartAllContainers(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		containers, err := dockClient.ListContainers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, c := range containers {
			if c.State != "running" {
				_ = dockClient.StartContainer(r.Context(), c.ID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}
}

// HandleStopAllContainers stops all running containers.
func HandleStopAllContainers(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		containers, err := dockClient.ListContainers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, c := range containers {
			if c.State == "running" {
				_ = dockClient.StopContainer(r.Context(), c.ID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}
}

// HandleDeleteContainer deletes a container by ID from Docker and SQLite DB. Returns structured JSON error if failed.
func HandleDeleteContainer(dockClient *Client, database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := r.URL.Query().Get("id")
		force := r.URL.Query().Get("force") == "true"

		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing container ID parameter"})
			return
		}

		var dockerErr error
		if dockClient != nil {
			dockerErr = dockClient.RemoveContainer(r.Context(), id, force)
		}

		if database != nil {
			_ = database.DeleteContainerByDockerID(r.Context(), id)
		}

		if dockerErr != nil {
			log.Printf("api: delete container %s error: %v", id, dockerErr)
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   dockerErr.Error(),
				"details": fmt.Sprintf("Failed to remove container '%s'. If it is running, stop it first or enable 'Force deletion'.", id),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Container deleted successfully"})
	}
}

// HandleDeleteVolume deletes a volume by name. Returns structured JSON error if failed.
func HandleDeleteVolume(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		name := r.URL.Query().Get("name")
		force := r.URL.Query().Get("force") == "true"

		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing volume name parameter"})
			return
		}

		if err := dockClient.RemoveVolume(r.Context(), name, force); err != nil {
			log.Printf("api: delete volume %s error: %v", name, err)
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   err.Error(),
				"details": fmt.Sprintf("Failed to remove volume '%s'. It may be in use by an active or stopped container.", name),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Volume deleted successfully"})
	}
}

// HandlePruneSystem prunes stopped containers and unused volumes.
func HandlePruneSystem(dockClient *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := dockClient.PruneSystem(r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "System pruned successfully"})
	}
}

// HandleListUserRepos fetches GitHub repositories for a given username or the authenticated token.
// If a github_token is stored in settings, it authenticates against /user/repos to return all public & private repos.
func HandleListUserRepos(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		username := strings.TrimSpace(r.URL.Query().Get("username"))

		var ghToken string
		var storedUser string
		if database != nil {
			if tok, err := database.GetSetting(r.Context(), "github_token"); err == nil && tok != "" {
				ghToken = strings.TrimSpace(tok)
			}
			if u, err := database.GetSetting(r.Context(), "github_username"); err == nil && u != "" {
				storedUser = strings.TrimSpace(u)
			}
		}

		if username == "" {
			username = storedUser
		}
		if username == "" {
			username = "VedantJJA"
		}

		// Build the API request URL
		var reqURL string
		if ghToken != "" {
			// Authenticated endpoint: returns all public & private repos for the authenticated PAT
			reqURL = "https://api.github.com/user/repos?sort=updated&per_page=100&type=all"
		} else if username != "" {
			// Public fallback for specified username
			reqURL = fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated&per_page=100", username)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"repos": []interface{}{}, "error": "No GitHub username or token configured in Settings."})
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), "GET", reqURL, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"repos": []interface{}{}, "error": err.Error()})
			return
		}
		req.Header.Set("User-Agent", "DevPnl-App")
		req.Header.Set("Accept", "application/vnd.github+json")
		if ghToken != "" {
			req.Header.Set("Authorization", "Bearer "+ghToken)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("api: fetch github repos error: %v", err)
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]interface{}{"repos": []interface{}{}, "error": "Failed to connect to GitHub API"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusUnauthorized {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"repos": []interface{}{},
					"error": "GitHub Personal Access Token is invalid or expired. Please update your token in Settings.",
				})
				return
			}
			w.WriteHeader(resp.StatusCode)
			json.NewEncoder(w).Encode(map[string]interface{}{"repos": []interface{}{}, "error": fmt.Sprintf("GitHub API returned status %d for user %s", resp.StatusCode, username)})
			return
		}

		var ghRepos []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			HTMLURL     string `json:"html_url"`
			Private     bool   `json:"private"`
			UpdatedAt   string `json:"updated_at"`
			Description string `json:"description"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&ghRepos); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"repos": []interface{}{}, "error": err.Error()})
			return
		}

		type RepoDTO struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			URL         string `json:"url"`
			Private     bool   `json:"private"`
			Updated     string `json:"updated"`
			Description string `json:"description"`
		}

		var dtoList []RepoDTO
		for _, repo := range ghRepos {
			updatedStr := repo.UpdatedAt
			if len(updatedStr) >= 10 {
				updatedStr = updatedStr[:10]
			}
			dtoList = append(dtoList, RepoDTO{
				ID:          repo.ID,
				Name:        repo.Name,
				FullName:    repo.FullName,
				URL:         repo.HTMLURL,
				Private:     repo.Private,
				Updated:     updatedStr,
				Description: repo.Description,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"repos":         dtoList,
			"authenticated": ghToken != "",
		})
	}
}

// HandleSettings handles GET and POST for /api/settings.
// GET returns all settings. POST accepts a JSON object of key-value pairs to upsert.
func HandleSettings(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			settings, err := database.GetAllSettings(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
				return
			}
			// Mask the token for security
			if tok, ok := settings["github_token"]; ok && len(tok) > 8 {
				settings["github_token"] = tok[:8] + "..." + tok[len(tok)-4:]
			}
			if _, ok := settings["routing_mode"]; !ok {
				settings["routing_mode"] = "path"
			}
			reqHost := r.Header.Get("X-Forwarded-Host")
			if reqHost == "" {
				reqHost = r.Host
			}
			if storedDomain, ok := settings["base_domain"]; !ok || storedDomain == "" || storedDomain == "localhost:8090" {
				if reqHost != "" {
					settings["base_domain"] = reqHost
				} else {
					settings["base_domain"] = "localhost:8090"
				}
			}
			json.NewEncoder(w).Encode(settings)

		case http.MethodPost:
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON"})
				return
			}
			for key, value := range body {
				if err := database.SetSetting(r.Context(), key, value); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
					return
				}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		}
	}
}

// MathRound rounds a float64 to specified decimal places.
func MathRound(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10.0
	}
	return float64(int(val*ratio+0.5)) / ratio
}
