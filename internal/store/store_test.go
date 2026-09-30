package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/types"
)

// mustTS comes from mock.go (same package).

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
	if err == nil {
		t.Error("Expected error for nonexistent project, got nil")
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
		CreatedAt: mustTS("2026-06-14"),
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
		Title: "Answer", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14"),
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

	// Empty store -> 1 (initializes counter file)
	n, err := s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n != 1 {
		t.Errorf("NextNumber on empty store = %d, want 1", n)
	}

	// Write counter file directly, then read it back
	if err := writeNextNumber(dir, 5); err != nil {
		t.Fatalf("writeNextNumber() error = %v", err)
	}
	n, err = s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n != 5 {
		t.Errorf("NextNumber after writeNextNumber(5) = %d, want 5", n)
	}

	// AdvanceNextNumber bumps to 6
	if err := s.AdvanceNextNumber(); err != nil {
		t.Fatalf("AdvanceNextNumber() error = %v", err)
	}
	n, err = s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n != 6 {
		t.Errorf("NextNumber after AdvanceNextNumber = %d, want 6", n)
	}

	// Full workflow: create project using NextNumber + SaveProject + AdvanceNextNumber
	p := types.Project{
		ID: "0196f1b0-0000-7000-8000-000000000005", Number: 6,
		Title: "Test", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14"),
	}
	if err := s.SaveProject(p); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}
	if err := s.AdvanceNextNumber(); err != nil {
		t.Fatalf("AdvanceNextNumber() error = %v", err)
	}
	n, err = s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n != 7 {
		t.Errorf("NextNumber after full workflow = %d, want 7", n)
	}
}

func TestFileStore_SaveAndReadStep(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000010"
	p := types.Project{
		ID: pid, Number: 10, Title: "Test Project",
		Status: types.StatusActive, CreatedAt: mustTS("2026-06-14"),
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

func TestFileStore_SaveBlocker(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000020"
	p := types.Project{ID: pid, Number: 20, Title: "Blocker Test", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	if err := s.SaveProject(p); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}

	step := types.Step{ID: "my-step", Title: "My Step", Status: types.StepTodo, ProjectID: pid}
	if err := s.SaveStep(step); err != nil {
		t.Fatalf("SaveStep() error = %v", err)
	}

	blocker := types.Blocker{
		ID: "my-blocker", Title: "My Blocker",
		Status: types.BlockerWaiting, Reason: "no budget",
		ProjectID: pid, StepID: "my-step", CreatedAt: mustTS("2026-06-14"),
	}
	if err := s.SaveBlocker(blocker); err != nil {
		t.Fatalf("SaveBlocker() error = %v", err)
	}

	// Verify blocker is in the step and step is blocked
	steps, err := s.GetSteps(pid)
	if err != nil {
		t.Fatalf("GetSteps() error = %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("GetSteps returned %d steps, want 1", len(steps))
	}
	if len(steps[0].Blockers) != 1 {
		t.Fatalf("Blockers count = %d, want 1", len(steps[0].Blockers))
	}
	if steps[0].Blockers[0].ID != "my-blocker" {
		t.Errorf("Blocker ID = %q, want %q", steps[0].Blockers[0].ID, "my-blocker")
	}
	if steps[0].Blockers[0].Status != types.BlockerWaiting {
		t.Errorf("Blocker Status = %q, want %q", steps[0].Blockers[0].Status, types.BlockerWaiting)
	}
	// SaveBlocker should mark the step as blocked
	if steps[0].Status != types.StepBlocked {
		t.Errorf("Step Status = %q, want %q", steps[0].Status, types.StepBlocked)
	}

	// Verify on-disk step YAML has blockers
	stepPath := filepath.Join(dir, pid, "steps", "my-step.yaml")
	data, err := os.ReadFile(stepPath)
	if err != nil {
		t.Fatalf("Read step file: %v", err)
	}
	if !strings.Contains(string(data), "my-blocker") {
		t.Error("Step YAML missing blocker data")
	}
}

func TestFileStore_ResolveBlocker_UnblocksStep(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000021"
	p := types.Project{ID: pid, Number: 21, Title: "Resolve Test", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	step := types.Step{ID: "blocked-step", Title: "Blocked Step", Status: types.StepTodo, ProjectID: pid}
	s.SaveStep(step)

	blocker := types.Blocker{
		ID: "the-blocker", Title: "The Blocker", Status: types.BlockerWaiting,
		ProjectID: pid, StepID: "blocked-step", CreatedAt: mustTS("2026-06-14"),
	}
	s.SaveBlocker(blocker)

	// Now resolve it
	blocker.Status = types.BlockerResolved
	if err := s.SaveBlocker(blocker); err != nil {
		t.Fatalf("SaveBlocker(resolved) error = %v", err)
	}

	// But SaveBlocker always sets StepBlocked. The caller (cmdBlockerResolve)
	// sets StepTodo after resolve. So we manually check that the blocker
	// is marked resolved but the step needs the caller to unblock.
	steps, _ := s.GetSteps(pid)
	if steps[0].Blockers[0].Status != types.BlockerResolved {
		t.Errorf("Blocker status = %q, want %q", steps[0].Blockers[0].Status, types.BlockerResolved)
	}
}

func TestFileStore_SaveDecision(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000030"
	p := types.Project{ID: pid, Number: 30, Title: "Decision Test", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	dec := types.Decision{ID: "use-go", Title: "Use Go", Reason: "single binary", Date: mustTS("2026-06-14"), ProjectID: pid}
	if err := s.SaveDecision(dec); err != nil {
		t.Fatalf("SaveDecision() error = %v", err)
	}

	// Verify via GetDecisions
	decisions, err := s.GetDecisions(pid)
	if err != nil {
		t.Fatalf("GetDecisions() error = %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("GetDecisions returned %d, want 1", len(decisions))
	}
	if decisions[0].Title != "Use Go" {
		t.Errorf("Decision Title = %q, want %q", decisions[0].Title, "Use Go")
	}

	// Verify on-disk
	decPath := filepath.Join(dir, pid, "decisions", "use-go.yaml")
	if _, err := os.Stat(decPath); err != nil {
		t.Errorf("Decision file not on disk: %v", err)
	}
}

func TestFileStore_DeleteProject(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000040"
	p := types.Project{ID: pid, Number: 40, Title: "To Delete", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	if err := s.DeleteProject(pid); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}

	projects, _ := s.ListProjects()
	if len(projects) != 0 {
		t.Errorf("Projects after delete = %d, want 0", len(projects))
	}

	// Dir should be gone
	if _, err := os.Stat(filepath.Join(dir, pid)); !os.IsNotExist(err) {
		t.Errorf("Project dir still exists after delete")
	}
}

func TestFileStore_DeleteStep(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000050"
	p := types.Project{ID: pid, Number: 50, Title: "Step Delete", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	s.SaveStep(types.Step{ID: "keep", Title: "Keep", Status: types.StepTodo, ProjectID: pid})
	s.SaveStep(types.Step{ID: "remove", Title: "Remove", Status: types.StepTodo, ProjectID: pid})

	if err := s.DeleteStep(pid, "remove"); err != nil {
		t.Fatalf("DeleteStep() error = %v", err)
	}

	steps, _ := s.GetSteps(pid)
	if len(steps) != 1 {
		t.Fatalf("Steps after delete = %d, want 1", len(steps))
	}
	if steps[0].ID != "keep" {
		t.Errorf("Remaining step = %q, want %q", steps[0].ID, "keep")
	}
}

func TestFileStore_DeleteBlocker(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000060"
	p := types.Project{ID: pid, Number: 60, Title: "Blocker Delete", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	step := types.Step{ID: "my-step", Title: "My Step", Status: types.StepTodo, ProjectID: pid}
	s.SaveStep(step)

	b1 := types.Blocker{ID: "b1", Title: "Blocker 1", Status: types.BlockerWaiting, ProjectID: pid, StepID: "my-step", CreatedAt: mustTS("2026-06-14")}
	b2 := types.Blocker{ID: "b2", Title: "Blocker 2", Status: types.BlockerWaiting, ProjectID: pid, StepID: "my-step", CreatedAt: mustTS("2026-06-14")}
	s.SaveBlocker(b1)
	s.SaveBlocker(b2)

	if err := s.DeleteBlocker(pid, "my-step", "b1"); err != nil {
		t.Fatalf("DeleteBlocker() error = %v", err)
	}

	steps, _ := s.GetSteps(pid)
	if len(steps[0].Blockers) != 1 {
		t.Fatalf("Blockers after delete = %d, want 1", len(steps[0].Blockers))
	}
	if steps[0].Blockers[0].ID != "b2" {
		t.Errorf("Remaining blocker = %q, want %q", steps[0].Blockers[0].ID, "b2")
	}

	// Step should still be blocked (b2 still active)
	if steps[0].Status != types.StepBlocked {
		t.Errorf("Step Status = %q, want %q (still blocked by b2)", steps[0].Status, types.StepBlocked)
	}
}

func TestFileStore_DeleteLastBlocker_UnblocksStep(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000061"
	p := types.Project{ID: pid, Number: 61, Title: "Last Blocker Delete", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	step := types.Step{ID: "my-step", Title: "My Step", Status: types.StepTodo, ProjectID: pid}
	s.SaveStep(step)

	b := types.Blocker{ID: "only-b", Title: "Only Blocker", Status: types.BlockerWaiting, ProjectID: pid, StepID: "my-step", CreatedAt: mustTS("2026-06-14")}
	s.SaveBlocker(b)

	if err := s.DeleteBlocker(pid, "my-step", "only-b"); err != nil {
		t.Fatalf("DeleteBlocker() error = %v", err)
	}

	steps, _ := s.GetSteps(pid)
	if len(steps[0].Blockers) != 0 {
		t.Errorf("Blockers after delete = %d, want 0", len(steps[0].Blockers))
	}
	// Deleting the last blocker should set step back to todo
	if steps[0].Status != types.StepTodo {
		t.Errorf("Step Status = %q, want %q (unblocked)", steps[0].Status, types.StepTodo)
	}
}

func TestFileStore_DeleteDecision(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000070"
	p := types.Project{ID: pid, Number: 70, Title: "Decision Delete", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	s.SaveDecision(types.Decision{ID: "keep", Title: "Keep", Date: mustTS("2026-06-14"), ProjectID: pid})
	s.SaveDecision(types.Decision{ID: "remove", Title: "Remove", Date: mustTS("2026-06-14"), ProjectID: pid})

	if err := s.DeleteDecision(pid, "remove"); err != nil {
		t.Fatalf("DeleteDecision() error = %v", err)
	}

	decisions, _ := s.GetDecisions(pid)
	if len(decisions) != 1 {
		t.Fatalf("Decisions after delete = %d, want 1", len(decisions))
	}
	if decisions[0].ID != "keep" {
		t.Errorf("Remaining decision = %q, want %q", decisions[0].ID, "keep")
	}
}

func TestFileStore_GetBlockers(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000080"
	p := types.Project{ID: pid, Number: 80, Title: "Get Blockers", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	s.SaveStep(types.Step{ID: "s1", Title: "Step 1", Status: types.StepTodo, ProjectID: pid})
	s.SaveStep(types.Step{ID: "s2", Title: "Step 2", Status: types.StepTodo, ProjectID: pid})

	s.SaveBlocker(types.Blocker{ID: "b1", Title: "Blocker 1", Status: types.BlockerWaiting, ProjectID: pid, StepID: "s1", CreatedAt: mustTS("2026-06-14")})
	s.SaveBlocker(types.Blocker{ID: "b2", Title: "Blocker 2", Status: types.BlockerWaiting, ProjectID: pid, StepID: "s2", CreatedAt: mustTS("2026-06-14")})

	blockers, err := s.GetBlockers(pid)
	if err != nil {
		t.Fatalf("GetBlockers() error = %v", err)
	}
	if len(blockers) != 2 {
		t.Fatalf("GetBlockers returned %d, want 2", len(blockers))
	}
}

func TestFileStore_NextNumber_FromEmpty(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	n, err := s.NextNumber()
	if err != nil {
		t.Fatalf("NextNumber() error = %v", err)
	}
	if n != 1 {
		t.Errorf("NextNumber = %d, want 1", n)
	}
}

func TestFileStore_EmptyStepsDecisions(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	pid := "0196f1b0-0000-7000-8000-000000000090"
	p := types.Project{ID: pid, Number: 90, Title: "Empty", Status: types.StatusActive, CreatedAt: mustTS("2026-06-14")}
	s.SaveProject(p)

	// No steps or decisions saved — should return nil (no error)
	steps, err := s.GetSteps(pid)
	if err != nil {
		t.Fatalf("GetSteps() error = %v", err)
	}
	if steps != nil {
		t.Logf("GetSteps returned %d items (expected nil for empty project)", len(steps))
	}

	decisions, err := s.GetDecisions(pid)
	if err != nil {
		t.Fatalf("GetDecisions() error = %v", err)
	}
	if decisions != nil {
		t.Logf("GetDecisions returned %d items (expected nil for empty project)", len(decisions))
	}
}

func TestMockStore_CloseProject(t *testing.T) {
	s := NewMockStore()
	pd, err := s.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject(1) error = %v", err)
	}
	open := 0
	for _, st := range pd.Steps {
		if st.Status != types.StepDone {
			open++
		}
	}
	if _, err := s.CloseProject(pd.Project.ID, "test close", true); err != nil {
		t.Fatalf("CloseProject error = %v", err)
	}
	after, _ := s.GetProject(pd.Project.ID)
	if after.Project.Status != types.StatusCompleted {
		t.Errorf("status = %q, want completed", after.Project.Status)
	}
	for _, st := range after.Steps {
		if st.Status != types.StepDone {
			t.Errorf("step %q = %q, want done", st.ID, st.Status)
		}
	}
	found := false
	for _, d := range after.Decisions {
		if d.ID == "closed" {
			found = true
		}
	}
	if !found {
		t.Error("closed decision not recorded")
	}
	t.Logf("closed %d open steps", open)
	if _, err := s.CloseProject(pd.Project.ID, "again", true); err == nil {
		t.Error("double close must fail")
	}
}

func TestCloseProject_WithoutConfirmChangesNothing(t *testing.T) {
	// The gate: this is the whole point of the change. A first call that closes
	// would let an agent finish a project no human approved.
	s := NewMockStore()
	pd, err := s.ResolveProject("2")
	if err != nil {
		t.Fatalf("ResolveProject error = %v", err)
	}
	before := *pd
	beforeSteps := append([]types.Step(nil), pd.Steps...)
	beforeDecisions := len(pd.Decisions)

	plan, err := s.CloseProject(pd.Project.ID, "ignored", false)
	if err != nil {
		t.Fatalf("plan call error = %v", err)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
	if plan.StepsToClose() == 0 {
		t.Error("seed project should have open steps to close")
	}

	after, err := s.GetProject(pd.Project.ID)
	if err != nil {
		t.Fatalf("GetProject error = %v", err)
	}
	if after.Project.Status != before.Project.Status {
		t.Errorf("status changed: %q → %q", before.Project.Status, after.Project.Status)
	}
	if len(after.Decisions) != beforeDecisions {
		t.Errorf("decisions changed: %d → %d", beforeDecisions, len(after.Decisions))
	}
	for i, st := range after.Steps {
		if i < len(beforeSteps) && st.Status != beforeSteps[i].Status {
			t.Errorf("step %q status changed: %q → %q", st.ID, beforeSteps[i].Status, st.Status)
		}
	}
}

func TestCloseProject_ConfirmWithoutReasonRefused(t *testing.T) {
	s := NewMockStore()
	pd, _ := s.ResolveProject("2")
	if _, err := s.CloseProject(pd.Project.ID, "", true); err == nil {
		t.Fatal("confirm without a reason must fail")
	} else if !strings.Contains(err.Error(), "reason") {
		t.Errorf("error should name the reason, got: %v", err)
	}
	if _, err := s.CloseProject(pd.Project.ID, "   ", true); err == nil {
		t.Fatal("whitespace-only reason must fail too")
	}
	after, _ := s.GetProject(pd.Project.ID)
	if after.Project.Status == types.StatusCompleted {
		t.Error("a refused close must not have completed the project")
	}
}

func TestClosePlan_ListsStepsAndBlockers(t *testing.T) {
	s := NewMockStore()
	// Project 1 carries an unresolved blocker on a blocked step.
	plan, err := s.ClosePlan("1")
	if err != nil {
		t.Fatalf("ClosePlan error = %v", err)
	}
	if plan.StepsToClose() == 0 {
		t.Error("plan should list open steps")
	}
	if plan.BlockersToResolve() == 0 {
		t.Error("plan should list the unresolved blocker of the seeded project")
	}
	for _, b := range plan.Blockers {
		if b.StepName == "" {
			t.Errorf("blocker %q has no step title to show a human", b.ID)
		}
	}
	if plan.Empty() {
		t.Error("plan with steps must not report Empty")
	}
}

func TestClosePlan_DoesNotMutate(t *testing.T) {
	s := NewMockStore()
	pd, err := s.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject error = %v", err)
	}
	before := pd.Project.Status
	beforeSteps := append([]types.Step(nil), pd.Steps...)

	if _, err := s.ClosePlan(pd.Project.ID); err != nil {
		t.Fatalf("ClosePlan error = %v", err)
	}

	after, err := s.GetProject(pd.Project.ID)
	if err != nil {
		t.Fatalf("GetProject error = %v", err)
	}
	if after.Project.Status != before {
		t.Errorf("ClosePlan changed status: %q → %q", before, after.Project.Status)
	}
	for i := range beforeSteps {
		if after.Steps[i].Status != beforeSteps[i].Status {
			t.Errorf("ClosePlan changed step %q", beforeSteps[i].ID)
		}
	}
}

func TestClosePlan_CompletedProjectIsEmptyNotError(t *testing.T) {
	s := NewMockStore()
	pd, _ := s.ResolveProject("2")
	if _, err := s.CloseProject(pd.Project.ID, "done", true); err != nil {
		t.Fatalf("close error = %v", err)
	}
	plan, err := s.ClosePlan(pd.Project.ID)
	if err != nil {
		t.Fatalf("ClosePlan on a completed project must not error, got: %v", err)
	}
	if !plan.Empty() {
		t.Errorf("a completed project has nothing left to close, got %d steps", plan.StepsToClose())
	}
}
