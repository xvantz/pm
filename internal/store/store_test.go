package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xvantz/pm/internal/types"
)

func TestMockStore_ListProjects(t *testing.T) {
	s := NewMockStore()
	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if len(projects) == 0 {
		t.Fatal("ListProjects() returned 0 projects")
	}

	// Should have AGH, PM, Navidrome, etc.
	titles := make(map[string]bool)
	for _, p := range projects {
		titles[p.Title] = true
	}

	if !titles["AdGuard Home"] {
		t.Error("Missing project: AdGuard Home")
	}
	if !titles["Project Memory (PM)"] {
		t.Error("Missing project: PM")
	}
	if !titles["Navidrome Music Collector"] {
		t.Error("Missing project: Navidrome")
	}
}

func TestMockStore_GetProject(t *testing.T) {
	s := NewMockStore()

	// Get by UUID
	pd, err := s.GetProject("0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a")
	if err != nil {
		t.Fatalf("GetProject(agh uuuid) error = %v", err)
	}
	if pd == nil {
		t.Fatal("GetProject() returned nil")
	}

	if pd.Project.Title != "AdGuard Home" {
		t.Errorf("Title = %q, want %q", pd.Project.Title, "AdGuard Home")
	}

	if len(pd.Steps) == 0 {
		t.Error("Steps empty for agh")
	}

	// Blockers are inside steps now
	blockersFound := 0
	for _, s := range pd.Steps {
		blockersFound += len(s.Blockers)
	}
	if blockersFound == 0 {
		t.Error("No blockers found in steps (router blocker expected)")
	}
}

func TestMockStore_ResolveProject(t *testing.T) {
	s := NewMockStore()

	// By number
	pd, err := s.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject(1) error = %v", err)
	}
	if pd == nil {
		t.Fatal("ResolveProject(1) returned nil")
	}
	if pd.Project.Title != "AdGuard Home" {
		t.Errorf("Project #1 title = %q, want %q", pd.Project.Title, "AdGuard Home")
	}

	// By UUID
	pd, err = s.ResolveProject("0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b")
	if err != nil {
		t.Fatalf("ResolveProject(PM uuid) error = %v", err)
	}
	if pd == nil {
		t.Fatal("ResolveProject(PM uuid) returned nil")
	}
	if pd.Project.Title != "Project Memory (PM)" {
		t.Errorf("Project title = %q, want %q", pd.Project.Title, "Project Memory (PM)")
	}
}

func TestMockStore_GetProject_NotFound(t *testing.T) {
	s := NewMockStore()
	pd, err := s.GetProject("nonexistent")
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if pd != nil {
		t.Error("Expected nil for nonexistent project")
	}
}

func TestMockStore_NextNumber(t *testing.T) {
	s := NewMockStore()
	n, err := s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n <= 6 {
		t.Errorf("NextNumber = %d, want > 6", n)
	}
}

func TestMockStore_SaveAndGetStep(t *testing.T) {
	s := NewMockStore()

	step := types.Step{
		ID: "test-step", Title: "Test Step",
		Status: types.StepTodo, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a",
	}

	if err := s.SaveStep(step); err != nil {
		t.Fatalf("SaveStep() error = %v", err)
	}

	steps, err := s.GetSteps("0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a")
	if err != nil {
		t.Fatalf("GetSteps() error = %v", err)
	}

	found := false
	for _, st := range steps {
		if st.ID == "test-step" {
			found = true
			if st.Title != "Test Step" {
				t.Errorf("Title = %q, want %q", st.Title, "Test Step")
			}
		}
	}
	if !found {
		t.Error("Saved step not found in GetSteps")
	}
}

func TestFileStore_CreateAndRead(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	p := types.Project{
		ID: "0196f1b0-0000-7000-8000-000000000001", Number: 1,
		Title: "Test Project", Goal: "Testing",
		Status: types.StatusActive, Tags: []string{"test"},
		CreatedAt: "2026-06-14",
	}

	if err := s.SaveProject(p); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}

	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("ListProjects returned %d projects, want 1", len(projects))
	}

	if projects[0].ID != "0196f1b0-0000-7000-8000-000000000001" {
		t.Errorf("Project ID = %q, want %q", projects[0].ID, "0196f1b0-0000-7000-8000-000000000001")
	}

	if projects[0].Number != 1 {
		t.Errorf("Project Number = %d, want 1", projects[0].Number)
	}
}

func TestFileStore_ResolveByNumber(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	p := types.Project{
		ID: "0196f1b0-0000-7000-8000-000000000001", Number: 42,
		Title: "Answer", Status: types.StatusActive, CreatedAt: "2026-06-14",
	}
	s.SaveProject(p)

	// Resolve by number
	pd, err := s.ResolveProject("42")
	if err != nil {
		t.Fatalf("ResolveProject(42) error = %v", err)
	}
	if pd == nil {
		t.Fatal("ResolveProject(42) returned nil")
	}
	if pd.Project.Title != "Answer" {
		t.Errorf("Title = %q, want %q", pd.Project.Title, "Answer")
	}

	// Resolve by UUID
	pd, err = s.ResolveProject("0196f1b0-0000-7000-8000-000000000001")
	if err != nil {
		t.Fatalf("ResolveProject(UUID) error = %v", err)
	}
	if pd == nil {
		t.Fatal("ResolveProject(UUID) returned nil")
	}
}

func TestFileStore_NextNumber(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	// Empty store -> 1
	n, err := s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n != 1 {
		t.Errorf("NextNumber on empty store = %d, want 1", n)
	}

	// Add a project with number 5
	p := types.Project{
		ID: "0196f1b0-0000-7000-8000-000000000005", Number: 5,
		Title: "Test", Status: types.StatusActive, CreatedAt: "2026-06-14",
	}
	s.SaveProject(p)

	n, err = s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n != 6 {
		t.Errorf("NextNumber = %d, want 6", n)
	}
}

func TestFileStore_SaveAndReadStep(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000010"
	p := types.Project{
		ID: pid, Number: 10, Title: "Test Project",
		Status: types.StatusActive, CreatedAt: "2026-06-14",
	}
	if err := s.SaveProject(p); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}

	step := types.Step{
		ID: "step1", Title: "First step",
		Status: types.StepDone, ProjectID: pid,
		Artifacts: []string{"docs/note.md"},
	}
	if err := s.SaveStep(step); err != nil {
		t.Fatalf("SaveStep() error = %v", err)
	}

	steps, err := s.GetSteps(pid)
	if err != nil {
		t.Fatalf("GetSteps() error = %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("GetSteps returned %d steps, want 1", len(steps))
	}

	if steps[0].Title != "First step" {
		t.Errorf("Step title = %q, want %q", steps[0].Title, "First step")
	}

	// Verify file exists on disk
	stepPath := filepath.Join(dir, pid, "steps", "step1.yaml")
	if _, err := os.Stat(stepPath); err != nil {
		t.Errorf("Step file not on disk: %v", err)
	}
}

func TestFileStore_GetProject_NotFound(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pd, err := s.GetProject("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent project, got nil")
	}
	if pd != nil {
		t.Error("Expected nil project data")
	}
}

func TestFileStore_ListProjects_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}
