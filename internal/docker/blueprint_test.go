package docker

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleBlueprintYAML = `
version: "1.0"
project: "my-monorepo-startup"
services:
  frontend:
    type: static
    source:
      repo: "https://github.com/username/my-monorepo.git"
      directory: "web"
    build:
      engine: "node"
  backend:
    type: web
    source:
      repo: "https://github.com/username/my-monorepo.git"
      directory: "api"
    build:
      engine: "dockerfile"
      dockerfile_path: "Dockerfile"
    deploy:
      port: 8080
  database:
    type: database
    image: "postgres:15"
    deploy:
      port: 5432
`

func TestParseBlueprint_Success(t *testing.T) {
	bp, err := ParseBlueprint([]byte(sampleBlueprintYAML))
	if err != nil {
		t.Fatalf("expected no parse error, got: %v", err)
	}

	if bp.Project != "my-monorepo-startup" {
		t.Errorf("expected project 'my-monorepo-startup', got: %s", bp.Project)
	}

	if len(bp.Services) != 3 {
		t.Fatalf("expected 3 services, got: %d", len(bp.Services))
	}

	// Verify frontend service
	fe, ok := bp.Services["frontend"]
	if !ok {
		t.Fatal("missing 'frontend' service")
	}
	if fe.Type != "static" {
		t.Errorf("expected type 'static', got: %s", fe.Type)
	}
	if fe.Source.Repo != "https://github.com/username/my-monorepo.git" {
		t.Errorf("expected repo URL, got: %s", fe.Source.Repo)
	}
	if fe.Source.Directory != "web" {
		t.Errorf("expected directory 'web', got: %s", fe.Source.Directory)
	}
	if fe.Build.Engine != "node" {
		t.Errorf("expected build engine 'node', got: %s", fe.Build.Engine)
	}

	// Verify backend service
	be, ok := bp.Services["backend"]
	if !ok {
		t.Fatal("missing 'backend' service")
	}
	if be.Type != "web" {
		t.Errorf("expected type 'web', got: %s", be.Type)
	}
	if be.Source.Directory != "api" {
		t.Errorf("expected directory 'api', got: %s", be.Source.Directory)
	}
	if be.Deploy.Port != 8080 {
		t.Errorf("expected port 8080, got: %d", be.Deploy.Port)
	}

	// Verify database service
	db, ok := bp.Services["database"]
	if !ok {
		t.Fatal("missing 'database' service")
	}
	if db.Type != "database" {
		t.Errorf("expected type 'database', got: %s", db.Type)
	}
	if db.Image != "postgres:15" {
		t.Errorf("expected image 'postgres:15', got: %s", db.Image)
	}
	if db.Deploy.Port != 5432 {
		t.Errorf("expected port 5432, got: %d", db.Deploy.Port)
	}
}

func TestParseBlueprint_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "Missing project name",
			yaml:    `version: "1.0"`,
			wantErr: "project name is required",
		},
		{
			name:    "No services defined",
			yaml:    `version: "1.0"\nproject: "test"`,
			wantErr: "at least one service",
		},
		{
			name: "Database without image",
			yaml: `
version: "1.0"
project: "test"
services:
  db:
    type: database
`,
			wantErr: "database type requires 'image'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBlueprint([]byte(tt.yaml))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
		})
	}
}

func TestCreateTarArchive(t *testing.T) {
	tempDir := t.TempDir()

	// Create dummy files inside sub-directory
	subDir := filepath.Join(tempDir, "web")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "index.html"), []byte("<h1>Hello Monorepo</h1>"), 0644); err != nil {
		t.Fatal(err)
	}

	tarBuf, err := createTarArchive(subDir)
	if err != nil {
		t.Fatalf("expected no tar error, got: %v", err)
	}

	if tarBuf.Len() == 0 {
		t.Fatal("expected non-empty tar buffer")
	}
}
