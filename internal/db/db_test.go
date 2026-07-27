package db

import (
	"context"
	"testing"
)

// testDB returns an in-memory database for testing.
func testDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:): %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// ---------- Projects --------------------------------------------------------

func TestProject_CRUD(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	// Create
	p := &Project{Name: "myapp", RepoURL: "https://github.com/test/myapp"}
	id, err := d.CreateProject(ctx, p)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if id < 1 {
		t.Fatalf("expected positive id, got %d", id)
	}

	// Get by ID
	got, err := d.GetProject(ctx, id)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != "myapp" {
		t.Errorf("expected name=myapp, got %q", got.Name)
	}
	if got.Status != "inactive" {
		t.Errorf("expected status=inactive, got %q", got.Status)
	}

	// Get by name
	got2, err := d.GetProjectByName(ctx, "myapp")
	if err != nil {
		t.Fatalf("GetProjectByName: %v", err)
	}
	if got2.ID != id {
		t.Errorf("expected ID=%d, got %d", id, got2.ID)
	}

	// Update
	got.Status = "running"
	got.ComposeYML = "version: '3'"
	if err := d.UpdateProject(ctx, got); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	updated, _ := d.GetProject(ctx, id)
	if updated.Status != "running" {
		t.Errorf("expected status=running after update, got %q", updated.Status)
	}

	// List
	d.CreateProject(ctx, &Project{Name: "otherapp"})
	list, err := d.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(list))
	}

	// Delete
	if err := d.DeleteProject(ctx, id); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	gone, _ := d.GetProject(ctx, id)
	if gone != nil {
		t.Error("expected nil after delete")
	}
}

func TestProject_DuplicateName(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	d.CreateProject(ctx, &Project{Name: "dup"})
	_, err := d.CreateProject(ctx, &Project{Name: "dup"})
	if err == nil {
		t.Fatal("expected error on duplicate project name")
	}
}

// ---------- Containers ------------------------------------------------------

func TestContainer_CRUD(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	// Create parent project first.
	pid, _ := d.CreateProject(ctx, &Project{Name: "proj"})

	c := &Container{
		ProjectID:   pid,
		ContainerID: "abc123",
		Name:        "web",
		Image:       "nginx:latest",
		Port:        8080,
	}
	cid, err := d.CreateContainer(ctx, c)
	if err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}

	// Get
	got, err := d.GetContainer(ctx, cid)
	if err != nil {
		t.Fatalf("GetContainer: %v", err)
	}
	if got.ContainerID != "abc123" {
		t.Errorf("expected container_id=abc123, got %q", got.ContainerID)
	}
	if got.Status != "created" {
		t.Errorf("expected status=created, got %q", got.Status)
	}

	// Get by Docker ID
	got2, err := d.GetContainerByDockerID(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetContainerByDockerID: %v", err)
	}
	if got2.ID != cid {
		t.Errorf("expected ID=%d, got %d", cid, got2.ID)
	}

	// Update status
	if err := d.UpdateContainerStatus(ctx, cid, "running"); err != nil {
		t.Fatalf("UpdateContainerStatus: %v", err)
	}
	updated, _ := d.GetContainer(ctx, cid)
	if updated.Status != "running" {
		t.Errorf("expected status=running, got %q", updated.Status)
	}

	// List by project
	d.CreateContainer(ctx, &Container{ProjectID: pid, ContainerID: "def456", Name: "db", Image: "postgres:16"})
	list, err := d.ListContainersByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListContainersByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(list))
	}

	// Delete
	if err := d.DeleteContainer(ctx, cid); err != nil {
		t.Fatalf("DeleteContainer: %v", err)
	}
	gone, _ := d.GetContainer(ctx, cid)
	if gone != nil {
		t.Error("expected nil after delete")
	}
}

func TestContainer_CascadeDelete(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	pid, _ := d.CreateProject(ctx, &Project{Name: "cascade-proj"})
	d.CreateContainer(ctx, &Container{ProjectID: pid, ContainerID: "x1", Name: "svc1"})
	d.CreateContainer(ctx, &Container{ProjectID: pid, ContainerID: "x2", Name: "svc2"})

	// Deleting the project should cascade-delete its containers.
	d.DeleteProject(ctx, pid)

	list, _ := d.ListContainersByProject(ctx, pid)
	if len(list) != 0 {
		t.Fatalf("expected 0 containers after cascade, got %d", len(list))
	}
}

// ---------- Domains ---------------------------------------------------------

func TestDomain_CRUD(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	pid, _ := d.CreateProject(ctx, &Project{Name: "domproj"})

	dom := &Domain{ProjectID: pid, FQDN: "app.example.com", TLS: true}
	did, err := d.CreateDomain(ctx, dom)
	if err != nil {
		t.Fatalf("CreateDomain: %v", err)
	}

	// Get
	got, err := d.GetDomain(ctx, did)
	if err != nil {
		t.Fatalf("GetDomain: %v", err)
	}
	if got.FQDN != "app.example.com" {
		t.Errorf("expected fqdn=app.example.com, got %q", got.FQDN)
	}
	if !got.TLS {
		t.Error("expected TLS=true")
	}

	// Get by FQDN
	got2, err := d.GetDomainByFQDN(ctx, "app.example.com")
	if err != nil {
		t.Fatalf("GetDomainByFQDN: %v", err)
	}
	if got2.ID != did {
		t.Errorf("expected ID=%d, got %d", did, got2.ID)
	}

	// DomainExists
	exists, err := d.DomainExists(ctx, "app.example.com")
	if err != nil {
		t.Fatalf("DomainExists: %v", err)
	}
	if !exists {
		t.Error("expected domain to exist")
	}

	notExists, _ := d.DomainExists(ctx, "nope.example.com")
	if notExists {
		t.Error("expected domain NOT to exist")
	}

	// List by project
	d.CreateDomain(ctx, &Domain{ProjectID: pid, FQDN: "api.example.com", TLS: false})
	list, err := d.ListDomainsByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListDomainsByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(list))
	}

	// Delete
	if err := d.DeleteDomain(ctx, did); err != nil {
		t.Fatalf("DeleteDomain: %v", err)
	}
	gone, _ := d.GetDomain(ctx, did)
	if gone != nil {
		t.Error("expected nil after delete")
	}
}

func TestDomain_CascadeDelete(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	pid, _ := d.CreateProject(ctx, &Project{Name: "cascade-dom"})
	d.CreateDomain(ctx, &Domain{ProjectID: pid, FQDN: "a.test.com"})
	d.CreateDomain(ctx, &Domain{ProjectID: pid, FQDN: "b.test.com"})

	d.DeleteProject(ctx, pid)

	list, _ := d.ListDomainsByProject(ctx, pid)
	if len(list) != 0 {
		t.Fatalf("expected 0 domains after cascade, got %d", len(list))
	}
}

func TestDomain_UniqueFQDN(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	pid, _ := d.CreateProject(ctx, &Project{Name: "uniq-dom"})
	d.CreateDomain(ctx, &Domain{ProjectID: pid, FQDN: "same.example.com"})
	_, err := d.CreateDomain(ctx, &Domain{ProjectID: pid, FQDN: "same.example.com"})
	if err == nil {
		t.Fatal("expected error on duplicate FQDN")
	}
}
