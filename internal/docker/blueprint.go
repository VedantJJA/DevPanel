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
	Version  string                   `yaml:"version"`
	Project  string                   `yaml:"project"`
	Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig represents a single application service or microservice in the blueprint.
type ServiceConfig struct {
	Type   string        `yaml:"type"`   // "web", "static", "database", "worker"
	Image  string        `yaml:"image"`  // pre-built image (e.g. "postgres:15")
	Source SourceConfig  `yaml:"source"` // monorepo source details
	Build  BuildConfig   `yaml:"build"`  // build engine options
	Deploy DeployConfig  `yaml:"deploy"` // ports, env vars, runtime settings
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

	// If no Dockerfile exists and engine is "node" or "static", generate a default production Dockerfile
	if _, err := os.Stat(fullDockerfilePath); os.IsNotExist(err) {
		if cfg.Build.Engine == "static" || cfg.Type == "static" {
			log.Printf("blueprint: generating static Nginx Dockerfile for %s", serviceName)
			if err := generateStaticDockerfile(fullDockerfilePath, cfg.Build.OutputDir, cfg.Build.Command); err != nil {
				return fmt.Errorf("generate static dockerfile: %w", err)
			}
		} else if cfg.Build.Engine == "node" {
			log.Printf("blueprint: generating Node.js Dockerfile for %s", serviceName)
			if err := generateNodeDockerfile(fullDockerfilePath); err != nil {
				return fmt.Errorf("generate node dockerfile: %w", err)
			}
		} else {
			return fmt.Errorf("Dockerfile not found at %s and engine %q is unsupported for auto-generation", fullDockerfilePath, cfg.Build.Engine)
		}
	}

	// Create in-memory tarball of the target directory
	tarBuf, err := createTarArchive(targetDir)
	if err != nil {
		return fmt.Errorf("create tar context: %w", err)
	}

	log.Printf("blueprint: tar context prepared for %s (%d bytes)", serviceName, tarBuf.Len())
	return o.buildImageViaAPI(ctx, tarBuf, imageName, dockerfilePath)
}

// buildImageViaAPI sends build context tarball to Docker Engine API.
func (o *BlueprintOrchestrator) buildImageViaAPI(ctx context.Context, tarBuf *bytes.Buffer, imageName string, dockerfilePath string) error {
	url := fmt.Sprintf("http://docker/build?t=%s&dockerfile=%s", imageName, dockerfilePath)
	req, err := http.NewRequestWithContext(ctx, "POST", url, tarBuf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-tar")

	resp, err := o.Client.http.Do(req)
	if err != nil {
		return fmt.Errorf("docker api build image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker api build failed (HTTP %d): %s", resp.StatusCode, body)
	}

	// Stream build output line by line
	scanner := bufio.NewScanner(resp.Body)
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
					log.Printf("blueprint-build [%s]: %s", imageName, msg)
				}
			}
			if buildMsg.Error != "" {
				return fmt.Errorf("docker build error: %s", buildMsg.Error)
			}
		}
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

	config := ContainerCreateConfig{
		Image:        imageName,
		Env:          envList,
		Cmd:          cmdList,
		ExposedPorts: exposedPorts,
		HostConfig: HostConfig{
			PortBindings: map[string][]PortBinding{
				portKey: {{HostPort: ""}}, // Auto-assign free host port
			},
		},
	}

	containerID, err := o.Client.CreateContainer(ctx, containerName, config)
	if err != nil {
		return "", fmt.Errorf("create container %s: %w", containerName, err)
	}

	if err := o.Client.StartContainer(ctx, containerID); err != nil {
		return "", fmt.Errorf("start container %s (%s): %w", containerName, containerID, err)
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
		return err
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
	if buildCmd == "" {
		buildCmd = "npm run build"
	}
	content := fmt.Sprintf(`FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci || npm install
COPY . .
RUN %s

FROM nginx:alpine
COPY --from=builder /app/%s /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
`, buildCmd, outputDir)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// generateNodeDockerfile writes a production-ready Node.js Dockerfile.
func generateNodeDockerfile(destPath string) error {
	content := `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production || npm install
COPY . .
EXPOSE 8080
CMD ["npm", "start"]
`
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
