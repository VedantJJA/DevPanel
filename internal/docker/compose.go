package docker

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

// ComposeUp runs `docker compose up -d` for the given compose file.
// projectDir is the directory containing the docker-compose.yml.
// Returns combined stdout+stderr output.
func ComposeUp(ctx context.Context, projectDir string) (string, error) {
	return composeCmd(ctx, projectDir, "up", "-d", "--remove-orphans")
}

// ComposeDown runs `docker compose down` for the given compose file.
func ComposeDown(ctx context.Context, projectDir string) (string, error) {
	return composeCmd(ctx, projectDir, "down", "--remove-orphans")
}

// ComposePull runs `docker compose pull` to fetch the latest images.
func ComposePull(ctx context.Context, projectDir string) (string, error) {
	return composeCmd(ctx, projectDir, "pull")
}

// ComposePS returns the container IDs of running services in the project.
func ComposePS(ctx context.Context, projectDir string) ([]string, error) {
	out, err := composeCmd(ctx, projectDir, "ps", "-q")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids, nil
}

// composeCmd executes a docker compose command in the given directory.
func composeCmd(ctx context.Context, projectDir string, args ...string) (string, error) {
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	fullArgs := append([]string{"compose", "-f", composePath}, args...)

	log.Printf("docker: exec docker %s", strings.Join(fullArgs, " "))

	cmd := exec.CommandContext(ctx, "docker", fullArgs...)
	cmd.Dir = projectDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker compose %s: %w\noutput: %s",
			strings.Join(args, " "), err, out)
	}

	return string(out), nil
}
