package apistore

import (
	"testing"

	"github.com/xvantz/pm/internal/types"
)

// The remote store must actually send the title: before this change
// SaveProject silently dropped it, so a rename lived only in the caller's
// memory and the daemon kept the old name.
func TestRemote_RenameProjectPropagates(t *testing.T) {
	st := newRemote(t)

	if err := st.SaveProject(types.Project{Title: "Old Name", Goal: "g"}); err != nil {
		t.Fatalf("SaveProject(new) error = %v", err)
	}
	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject(1) error = %v", err)
	}

	p := pd.Project
	p.Title = "New Name"
	if err := st.SaveProject(p); err != nil {
		t.Fatalf("SaveProject(rename) error = %v", err)
	}

	got, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject(1) after rename error = %v", err)
	}
	if got.Project.Title != "New Name" {
		t.Errorf("title = %q, want %q", got.Project.Title, "New Name")
	}
	if got.Project.Goal != "g" {
		t.Errorf("goal = %q, want untouched %q", got.Project.Goal, "g")
	}
}

func TestRemote_RenameStepPropagates(t *testing.T) {
	st := newRemote(t)

	if err := st.SaveProject(types.Project{Title: "P"}); err != nil {
		t.Fatalf("SaveProject error = %v", err)
	}
	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject error = %v", err)
	}
	now := types.NowTimestamp()
	step := types.Step{
		ID: "old-step", Title: "Old Step", Status: types.StepTodo,
		ProjectID: pd.Project.ID, CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveStep(step); err != nil {
		t.Fatalf("SaveStep(new) error = %v", err)
	}

	step.Title = "New Step Title"
	if err := st.SaveStep(step); err != nil {
		t.Fatalf("SaveStep(rename) error = %v", err)
	}

	steps, err := st.GetSteps(pd.Project.ID)
	if err != nil {
		t.Fatalf("GetSteps error = %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(steps))
	}
	if steps[0].Title != "New Step Title" {
		t.Errorf("title = %q, want %q", steps[0].Title, "New Step Title")
	}
	if steps[0].ID != "old-step" {
		t.Errorf("id moved on rename: %q", steps[0].ID)
	}
}
