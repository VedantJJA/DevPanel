package docker

import (
	"encoding/json"
	"log"
	"net/http"

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

		var dtoList []ContainerDTO
		for _, c := range containers {
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

			// Get one-shot stats for running container if available
			var cpu float64
			var mem float64
			if status == "running" {
				stats, err := dockClient.Stats(r.Context(), c.ID)
				if err == nil && stats != nil {
					cpu = stats.CPUPercent
					mem = stats.MemUsageMB
				}
			}

			dtoList = append(dtoList, ContainerDTO{
				ID:         shortID,
				Name:       name,
				Image:      c.Image,
				Status:     status,
				Port:       FormatPorts(c.Ports),
				CPUPercent: MathRound(cpu, 1),
				MemoryMB:   MathRound(mem, 1),
				Uptime:     c.Status,
			})
		}

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

// MathRound rounds a float64 to specified decimal places.
func MathRound(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10.0
	}
	return float64(int(val*ratio+0.5)) / ratio
}
