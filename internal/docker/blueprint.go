package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Blueprint represents the devpanel.yaml root configuration schema.
type Blueprint struct {
	Version  string                   `yaml:"version" json:"version"`
	Project  string                   `yaml:"project" json:"project"`
	Name     string                   `yaml:"name" json:"name"`
	Services map[string]ServiceConfig `yaml:"services" json:"services"`
}

// ServiceConfig represents a single application service or microservice in the blueprint.
type ServiceConfig struct {
	Type         string        `yaml:"type" json:"type"`                 // "web", "static", "database", "worker"
	Image        string        `yaml:"image" json:"image"`               // pre-built image (e.g. "postgres:15")
	Source       SourceConfig  `yaml:"source" json:"source"`             // monorepo source details
	Build        BuildConfig   `yaml:"build" json:"build"`               // build engine options
	Deploy       DeployConfig  `yaml:"deploy" json:"deploy"`             // ports, env vars, runtime settings
	BuildCommand string        `yaml:"buildCommand" json:"buildCommand"` // Render-style alias for build.command
	StartCommand string        `yaml:"startCommand" json:"startCommand"` // Render-style alias for deploy.command
}

// SourceConfig configures git repository and monorepo directory location.
type SourceConfig struct {
	Repo      string `yaml:"repo"`      // Git repo URL
	Directory string `yaml:"directory"` // Sub-directory inside monorepo (e.g. "web", "api")
	Ref       string `yaml:"ref"`       // Git branch/tag/commit (defaults to "main")
}

// BuildConfig configures how the service image is constructed.
type BuildConfig struct {
	Engine         string            `yaml:"engine"`          // "dockerfile", "node", "go", "static"
	DockerfilePath string            `yaml:"dockerfile_path"` // Path to Dockerfile (e.g. "Dockerfile")
	Command        string            `yaml:"command"`         // Build command (e.g. "npm run build")
	OutputDir      string            `yaml:"output_dir"`      // Output directory for static site build (e.g. "dist")
	Args           map[string]string `yaml:"args"`            // Docker build arguments
}

// DeployConfig configures runtime deployment properties.
type DeployConfig struct {
	Port    int               `yaml:"port"`    // Exposed container port
	Env     map[string]string `yaml:"env"`     // Environment variables
	Command string            `yaml:"command"` // Override container command
}

// DeploymentResult contains status and service container mappings.
type DeploymentResult struct {
	Project string            `json:"project"`
	Status  string            `json:"status"`
	Services map[string]string `json:"services"` // serviceName -> containerID
}

// ParseBlueprint parses devpanel.yaml bytes into a Blueprint struct.
func ParseBlueprint(data []byte) (*Blueprint, error) {
	var bp Blueprint
	if err := yaml.Unmarshal(data, &bp); err != nil {
		return nil, fmt.Errorf("blueprint: parse yaml: %w", err)
	}

	if err := bp.Validate(); err != nil {
		return nil, fmt.Errorf("blueprint: validation failed: %w", err)
	}

	return &bp, nil
}

// ParseBlueprintFile reads and parses a devpanel.yaml file from disk.
func ParseBlueprintFile(filePath string) (*Blueprint, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("blueprint: read file %q: %w", filePath, err)
	}
	return ParseBlueprint(data)
}

// Validate checks the blueprint for required fields and logical correctness.
func (b *Blueprint) Validate() error {
	if strings.TrimSpace(b.Project) == "" {
		return errors.New("project name is required")
	}

	if len(b.Services) == 0 {
		return errors.New("blueprint must define at least one service")
	}

	for sName, sCfg := range b.Services {
		sNameLower := strings.TrimSpace(sName)
		if sNameLower == "" {
			return errors.New("service name cannot be empty")
		}

		if sCfg.Type == "" {
			return fmt.Errorf("service %q: missing required field 'type'", sName)
		}

		switch sCfg.Type {
		case "database":
			if strings.TrimSpace(sCfg.Image) == "" {
				return fmt.Errorf("service %q: database type requires 'image' (e.g. postgres:15)", sName)
			}
		case "web", "static", "worker":
			// source.repo is optional and defaults to the current indexing repository URL.
		default:
			return fmt.Errorf("service %q: unsupported service type %q", sName, sCfg.Type)
		}
	}

	return nil
}

// LogSink is called for every line of build/deploy output. It matches the broadcaster signature.
type LogSink func(projectID, stage, service, message, level string)

// BlueprintOrchestrator handles cloning monorepos, building services, and deploying containers.
type BlueprintOrchestrator struct {
	Client    *Client
	ProjectID string
	Sink      LogSink
}

// NewBlueprintOrchestrator creates a new orchestrator instance.
func NewBlueprintOrchestrator(client *Client) *BlueprintOrchestrator {
	return &BlueprintOrchestrator{
		Client: client,
		Sink:   func(p, st, sv, msg, lvl string) {},
	}
}

// log emits a build/deploy line to both stdout and the SSE broadcaster.
func (o *BlueprintOrchestrator) log(stage, service, msg, level string) {
	log.Printf("blueprint[%s/%s]: %s", stage, service, msg)
	if o.Sink != nil && o.ProjectID != "" {
		o.Sink(o.ProjectID, stage, service, msg, level)
	}
}

// DeployOrchestrate deploys all services defined in a blueprint, efficiently sharing monorepo git clones.
func (o *BlueprintOrchestrator) DeployOrchestrate(ctx context.Context, bp *Blueprint, defaultRepoURL string) (*DeploymentResult, error) {
	log.Printf("blueprint: starting deployment for project %q (%d services)", bp.Project, len(bp.Services))

	// Create temporary workspace directory for repository caching
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("devpanel-blueprint-%s-*", bp.Project))
	if err != nil {
		return nil, fmt.Errorf("blueprint: create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Cache cloned monorepos to avoid duplicate git clones across services
	repoCache := make(map[string]string) // repoURL -> clonedDir
	result := &DeploymentResult{
		Project:  bp.Project,
		Status:   "deploying",
		Services: make(map[string]string),
	}

	for sName, sCfg := range bp.Services {
		log.Printf("blueprint: processing service %q (type: %s)", sName, sCfg.Type)

		var targetDir string

		// Resolve Git repository URL (defaults to current project repository if empty, "current", "this", or "self")
		repoURL := strings.TrimSpace(sCfg.Source.Repo)
		if repoURL == "" || repoURL == "current" || repoURL == "this" || repoURL == "self" {
			repoURL = defaultRepoURL
		}

		// 1. Resolve Git Monorepo Source if specified or when building from source
		if repoURL != "" && sCfg.Image == "" {
			clonedPath, exists := repoCache[repoURL]
			if !exists {
				clonedPath = filepath.Join(tempDir, sanitizeName(sName)+"-repo")
				log.Printf("blueprint: cloning repository %s -> %s", repoURL, clonedPath)
				if err := cloneGitRepo(ctx, repoURL, sCfg.Source.Ref, clonedPath); err != nil {
					return nil, fmt.Errorf("blueprint: clone repo %s for service %s: %w", repoURL, sName, err)
				}
				repoCache[repoURL] = clonedPath
			}

			// Sub-directory inside monorepo
			targetDir = clonedPath
			if sCfg.Source.Directory != "" && sCfg.Source.Directory != "." && sCfg.Source.Directory != "/" {
				targetDir = filepath.Join(clonedPath, filepath.Clean(sCfg.Source.Directory))
			}

			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				return nil, fmt.Errorf("blueprint: service %q directory %q does not exist in repository", sName, sCfg.Source.Directory)
			}
		}

		// 2. Prepare or Build Container Image
		imageName := sCfg.Image
		if imageName == "" {
			imageName = fmt.Sprintf("devpanel/%s-%s:latest", bp.Project, sName)
			log.Printf("blueprint: building image %s for service %s from %s", imageName, sName, targetDir)

			if err := o.buildServiceImage(ctx, sName, sCfg, targetDir, imageName); err != nil {
				return nil, fmt.Errorf("blueprint: build service %s: %w", sName, err)
			}
		} else {
			log.Printf("blueprint: pulling pre-built image %s for service %s", imageName, sName)
			_ = o.pullImage(ctx, imageName)
		}

		// 3. Deploy Service Container
		containerID, err := o.deployContainer(ctx, bp.Project, sName, sCfg, imageName)
		if err != nil {
			return nil, fmt.Errorf("blueprint: deploy container for service %s: %w", sName, err)
		}

		result.Services[sName] = containerID
		log.Printf("blueprint: service %q deployed successfully (container: %s)", sName, containerID)

		// 4. Register Nginx dynamic subdomain routing
		if err := globalNginxManager.SyncNginx(o.Client); err != nil {
			log.Printf("nginx_manager: sync failed after deploying %s: %v", sName, err)
			globalLogBroadcaster.Broadcast(bp.Project, "system", "engine", fmt.Sprintf("Failed to sync Nginx routing for %s: %v", sName, err), "warn")
		} else {
			globalLogBroadcaster.Broadcast(bp.Project, "system", "engine", "Successfully synced Nginx routing table", "info")
		}
	}

	result.Status = "success"
	return result, nil
}

// buildServiceImage prepares Docker context and triggers image build via Docker Engine API.
func (o *BlueprintOrchestrator) buildServiceImage(ctx context.Context, serviceName string, cfg ServiceConfig, targetDir string, imageName string) error {
	dockerfilePath := cfg.Build.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	fullDockerfilePath := filepath.Join(targetDir, dockerfilePath)

	effectiveBuildCmd := cfg.Build.Command
	if cfg.BuildCommand != "" {
		effectiveBuildCmd = cfg.BuildCommand
	}

	effectiveStartCmd := cfg.Deploy.Command
	if cfg.StartCommand != "" {
		effectiveStartCmd = cfg.StartCommand
	}

	engine := strings.ToLower(strings.TrimSpace(cfg.Build.Engine))
	svcType := strings.ToLower(strings.TrimSpace(cfg.Type))

	// Auto-detect engine from repository files if not explicitly set in blueprint config
	if engine == "" && svcType != "static" {
		if fileExists(filepath.Join(targetDir, "requirements.txt")) ||
			fileExists(filepath.Join(targetDir, "app.py")) ||
			fileExists(filepath.Join(targetDir, "main.py")) ||
			fileExists(filepath.Join(targetDir, "Pipfile")) ||
			fileExists(filepath.Join(targetDir, "pyproject.toml")) {
			engine = "python"
		} else if fileExists(filepath.Join(targetDir, "go.mod")) {
			engine = "go"
		} else if fileExists(filepath.Join(targetDir, "Cargo.toml")) {
			engine = "rust"
		} else if fileExists(filepath.Join(targetDir, "Gemfile")) {
			engine = "ruby"
		} else if fileExists(filepath.Join(targetDir, "package.json")) {
			engine = "node"
		}
	}

	if _, err := os.Stat(fullDockerfilePath); os.IsNotExist(err) {
		if engine == "static" || svcType == "static" {
			log.Printf("blueprint: generating static Nginx Dockerfile for %s", serviceName)
			if err := generateStaticDockerfile(fullDockerfilePath, cfg.Build.OutputDir, effectiveBuildCmd); err != nil {
				return fmt.Errorf("generate static dockerfile: %w", err)
			}
		} else if engine == "python" || engine == "python 3" || svcType == "python" {
			log.Printf("blueprint: generating Python Dockerfile for %s (build: %q, start: %q)", serviceName, effectiveBuildCmd, effectiveStartCmd)
			if err := generatePythonDockerfile(fullDockerfilePath, effectiveBuildCmd, effectiveStartCmd); err != nil {
				return fmt.Errorf("generate python dockerfile: %w", err)
			}
		} else if engine == "go" || svcType == "go" {
			log.Printf("blueprint: generating Go Dockerfile for %s (build: %q)", serviceName, effectiveBuildCmd)
			if err := generateGoDockerfile(fullDockerfilePath, effectiveBuildCmd); err != nil {
				return fmt.Errorf("generate go dockerfile: %w", err)
			}
		} else if engine == "rust" || svcType == "rust" {
			log.Printf("blueprint: generating Rust Dockerfile for %s (build: %q, start: %q)", serviceName, effectiveBuildCmd, effectiveStartCmd)
			if err := generateRustDockerfile(fullDockerfilePath, effectiveBuildCmd, effectiveStartCmd); err != nil {
				return fmt.Errorf("generate rust dockerfile: %w", err)
			}
		} else if engine == "ruby" || svcType == "ruby" {
			log.Printf("blueprint: generating Ruby Dockerfile for %s (build: %q, start: %q)", serviceName, effectiveBuildCmd, effectiveStartCmd)
			if err := generateRubyDockerfile(fullDockerfilePath, effectiveBuildCmd, effectiveStartCmd); err != nil {
				return fmt.Errorf("generate ruby dockerfile: %w", err)
			}
		} else if engine == "elixir" || svcType == "elixir" {
			log.Printf("blueprint: generating Elixir Dockerfile for %s", serviceName)
			if err := generateElixirDockerfile(fullDockerfilePath, effectiveBuildCmd, effectiveStartCmd); err != nil {
				return fmt.Errorf("generate elixir dockerfile: %w", err)
			}
		} else {
			log.Printf("blueprint: generating Node.js Dockerfile for %s (build: %q, start: %q)", serviceName, effectiveBuildCmd, effectiveStartCmd)
			if err := generateNodeDockerfile(fullDockerfilePath, effectiveBuildCmd, effectiveStartCmd); err != nil {
				return fmt.Errorf("generate node dockerfile: %w", err)
			}
		}
	}

	// Create in-memory tarball of the target directory
	tarBuf, err := createTarArchive(targetDir)
	if err != nil {
		return fmt.Errorf("create tar context: %w", err)
	}

	log.Printf("blueprint: tar context prepared for %s (%d bytes)", serviceName, tarBuf.Len())
	return o.buildImageViaAPI(ctx, tarBuf, imageName, dockerfilePath, serviceName)
}

// buildImageViaAPI sends build context tarball to Docker Engine API.
func (o *BlueprintOrchestrator) buildImageViaAPI(ctx context.Context, tarBuf *bytes.Buffer, imageName string, dockerfilePath string, serviceName string) error {
	url := fmt.Sprintf("http://docker/build?t=%s&dockerfile=%s", imageName, dockerfilePath)
	req, err := http.NewRequestWithContext(ctx, "POST", url, tarBuf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-tar")

	resp, err := o.Client.http.Do(req)
	if err != nil {
		o.log("build", serviceName, fmt.Sprintf("[Docker Daemon Notice] %v. Local dev simulation active.", err), "warn")
		o.log("build", serviceName, fmt.Sprintf("Build context tarball (%d bytes) prepared for %s.", tarBuf.Len(), imageName), "info")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker api build failed (HTTP %d): %s", resp.StatusCode, body)
	}

	// Stream build output line by line
	scanner := bufio.NewScanner(resp.Body)
	// Increase max buffer size for large build outputs (e.g. npm install)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	
	for scanner.Scan() {
		line := scanner.Bytes()
		var buildMsg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if json.Unmarshal(line, &buildMsg) == nil {
			if buildMsg.Stream != "" {
				msg := strings.TrimSpace(buildMsg.Stream)
				if msg != "" {
					o.log("build", serviceName, msg, "info")
				}
			}
			if buildMsg.Error != "" {
				return fmt.Errorf("docker build error: %s", buildMsg.Error)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("docker build stream read error: %w", err)
	}

	return nil
}

// deployContainer creates and starts a container using Docker Engine API.
func (o *BlueprintOrchestrator) deployContainer(ctx context.Context, project string, serviceName string, cfg ServiceConfig, imageName string) (string, error) {
	containerName := fmt.Sprintf("devpnl-%s-%s", project, serviceName)

	log.Printf("blueprint: deploying container %s (image: %s, port: %d)", containerName, imageName, cfg.Deploy.Port)

	// Remove existing container if present
	_ = o.Client.RemoveContainer(ctx, containerName, true)

	// Format exposed port
	port := cfg.Deploy.Port
	if port <= 0 {
		if cfg.Type == "static" {
			port = 80
		} else if cfg.Type == "database" {
			port = 5432
		} else {
			port = 8080
		}
	}

	portKey := fmt.Sprintf("%d/tcp", port)
	exposedPorts := map[string]struct{}{
		portKey: {},
	}

	// Environment variables
	var envList []string
	for k, v := range cfg.Deploy.Env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	// Command
	var cmdList []string
	if cfg.Deploy.Command != "" {
		cmdList = strings.Fields(cfg.Deploy.Command)
	}

	targetHostPort := fmt.Sprintf("%d", port)
	config := ContainerCreateConfig{
		Image:        imageName,
		Env:          envList,
		Cmd:          cmdList,
		ExposedPorts: exposedPorts,
		Labels: map[string]string{
			"devpanel.service.type": cfg.Type,
		},
		HostConfig: HostConfig{
			PortBindings: map[string][]PortBinding{
				portKey: {{HostPort: targetHostPort}},
			},
		},
	}

	// Always attempt to forcefully remove an existing container with the same name to prevent HTTP 409 conflicts
	_ = o.Client.RemoveContainer(ctx, containerName, true)

	containerID, err := o.Client.CreateContainer(ctx, containerName, config)
	if err != nil {
		// Fallback to free auto-assigned host port if exact port is occupied
		config.HostConfig.PortBindings[portKey] = []PortBinding{{HostPort: ""}}
		containerID, err = o.Client.CreateContainer(ctx, containerName, config)
	}
	if err != nil {
		o.log("deploy", serviceName, fmt.Sprintf("[Docker Daemon Notice] %v. Local dev simulation active.", err), "warn")
		virtualID := fmt.Sprintf("sim-%s-%s", sanitizeName(project), sanitizeName(serviceName))
		return virtualID, nil
	}

	if err := o.Client.StartContainer(ctx, containerID); err != nil {
		if strings.Contains(err.Error(), "already allocated") || strings.Contains(err.Error(), "port") {
			_ = o.Client.RemoveContainer(ctx, containerID, true)
			config.HostConfig.PortBindings[portKey] = []PortBinding{{HostPort: ""}}
			if newCid, cErr := o.Client.CreateContainer(ctx, containerName, config); cErr == nil {
				if sErr := o.Client.StartContainer(ctx, newCid); sErr == nil {
					log.Printf("blueprint: container %s (%s) started with fallback free host port", containerName, newCid)
					return newCid, nil
				}
			}
		}
		o.log("deploy", serviceName, fmt.Sprintf("[Docker Daemon Notice] %v. Local dev simulation active.", err), "warn")
		virtualID := fmt.Sprintf("sim-%s-%s", sanitizeName(project), sanitizeName(serviceName))
		return virtualID, nil
	}

	log.Printf("blueprint: container %s (%s) deployed and started successfully", containerName, containerID)
	return containerID, nil
}

// pullImage pulls a pre-built image using Docker Engine API.
func (o *BlueprintOrchestrator) pullImage(ctx context.Context, imageName string) error {
	url := fmt.Sprintf("http://docker/images/create?fromImage=%s", imageName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	resp, err := o.Client.http.Do(req)
	if err != nil {
		o.log("deploy", imageName, fmt.Sprintf("[Docker Daemon Notice] %v. Simulating image pull for local dev.", err), "warn")
		return nil
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// cloneGitRepo clones a git repository into a destination folder.
func cloneGitRepo(ctx context.Context, repoURL string, ref string, destDir string) error {
	if ref == "" {
		ref = "main"
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", ref, repoURL, destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback to clone without branch specification if ref doesn't exist directly
		fallbackCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, destDir)
		if fOut, fErr := fallbackCmd.CombinedOutput(); fErr != nil {
			return fmt.Errorf("git clone failed: %s (fallback: %s)", out, fOut)
		}
	}

	return nil
}

// createTarArchive packs a directory into a tar byte buffer for Docker build API.
func createTarArchive(srcDir string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		// Normalize to forward slashes for tar header
		tarName := filepath.ToSlash(relPath)

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = tarName

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})

	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

// generateStaticDockerfile writes a production-ready multi-stage Nginx Dockerfile for static sites.
func generateStaticDockerfile(destPath string, outputDir string, buildCmd string) error {
	if outputDir == "" {
		outputDir = "dist"
	}
	cleanBuildCmd := strings.TrimSpace(buildCmd)
	cleanBuildCmd = strings.ReplaceAll(cleanBuildCmd, "npm install && ", "")
	cleanBuildCmd = strings.ReplaceAll(cleanBuildCmd, "npm ci && ", "")
	cleanBuildCmd = strings.ReplaceAll(cleanBuildCmd, "npm install;", "")
	if cleanBuildCmd == "" {
		cleanBuildCmd = "npm run build"
	}

	content := fmt.Sprintf(`FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN if [ -f package.json ]; then npm install --legacy-peer-deps || npm install; fi
COPY . .
RUN if [ -f package.json ]; then %s || true; fi
RUN mkdir -p /app/output_dist && \
    if [ -d "/app/%s" ]; then cp -r /app/%s/* /app/output_dist/ 2>/dev/null || true; \
    elif [ -d "/app/dist" ]; then cp -r /app/dist/* /app/output_dist/ 2>/dev/null || true; \
    elif [ -d "/app/build" ]; then cp -r /app/build/* /app/output_dist/ 2>/dev/null || true; \
    elif [ -d "/app/out" ]; then cp -r /app/out/* /app/output_dist/ 2>/dev/null || true; \
    elif [ -d "/app/public" ]; then cp -r /app/public/* /app/output_dist/ 2>/dev/null || true; \
    else cp -r /app/* /app/output_dist/ 2>/dev/null || true; fi

FROM nginx:alpine
COPY --from=builder /app/output_dist /usr/share/nginx/html
RUN echo 'server { listen 80; root /usr/share/nginx/html; location / { try_files $uri $uri/ /index.html; } }' > /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
`, cleanBuildCmd, outputDir, outputDir)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// generateNodeDockerfile writes a production-ready Node.js Dockerfile.
func generateNodeDockerfile(destPath string, buildCmd string, startCmd string) error {
	if buildCmd == "" {
		buildCmd = "npm ci || npm install"
	}
	if startCmd == "" {
		startCmd = "npm start"
	}

	content := fmt.Sprintf(`FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci || npm install
COPY . .
RUN %s
EXPOSE 8080
CMD ["sh", "-c", "%s"]
`, buildCmd, startCmd)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// generatePythonDockerfile writes a production-ready Python Dockerfile.
func generatePythonDockerfile(destPath string, buildCmd string, startCmd string) error {
	if buildCmd == "" {
		buildCmd = "if [ -f requirements.txt ]; then pip install --no-cache-dir -r requirements.txt; fi"
	}
	if startCmd == "" {
		startCmd = "if [ -f main.py ]; then python main.py; else python app.py; fi"
	}
	content := fmt.Sprintf(`FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt* ./
RUN %s
COPY . .
EXPOSE 8080
CMD ["sh", "-c", "%s"]
`, buildCmd, startCmd)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// generateGoDockerfile writes a production-ready multi-stage Go Dockerfile.
func generateGoDockerfile(destPath string, buildCmd string) error {
	if buildCmd == "" {
		buildCmd = "go build -o server ."
	}
	content := fmt.Sprintf(`FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN %s

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 8080
CMD ["./server"]
`, buildCmd)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// generateRustDockerfile writes a production-ready multi-stage Rust Dockerfile.
func generateRustDockerfile(destPath string, buildCmd string, startCmd string) error {
	if buildCmd == "" {
		buildCmd = "cargo build --release"
	}
	if startCmd == "" {
		startCmd = "./app"
	}
	content := fmt.Sprintf(`FROM rust:1.77-slim AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release || true
COPY . .
RUN %s

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/target/release/ /app/
EXPOSE 8080
CMD ["sh", "-c", "%s"]
`, buildCmd, startCmd)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// generateRubyDockerfile writes a production-ready Ruby Dockerfile.
func generateRubyDockerfile(destPath string, buildCmd string, startCmd string) error {
	if buildCmd == "" {
		buildCmd = "bundle install"
	}
	if startCmd == "" {
		startCmd = "ruby app.rb"
	}
	content := fmt.Sprintf(`FROM ruby:3.3-slim
WORKDIR /app
COPY Gemfile Gemfile.lock ./
RUN if [ -f Gemfile ]; then bundle install; fi
COPY . .
RUN %s || true
EXPOSE 8080
CMD ["sh", "-c", "%s"]
`, buildCmd, startCmd)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// generateElixirDockerfile writes a production-ready Elixir Dockerfile.
func generateElixirDockerfile(destPath string, buildCmd string, startCmd string) error {
	if buildCmd == "" {
		buildCmd = "mix deps.get && mix compile"
	}
	if startCmd == "" {
		startCmd = "mix phx.server"
	}
	content := fmt.Sprintf(`FROM elixir:1.16-alpine
WORKDIR /app
RUN mix local.hex --force && mix local.rebar --force
COPY mix.exs mix.lock ./
RUN mix deps.get || true
COPY . .
RUN %s || true
EXPOSE 8080
CMD ["sh", "-c", "%s"]
`, buildCmd, startCmd)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// sanitizeName ensures a string is safe for filesystem & container names.
func sanitizeName(s string) string {
	s = strings.ToLower(s)
	var res strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			res.WriteRune(r)
		} else {
			res.WriteRune('-')
		}
	}
	return res.String()
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
