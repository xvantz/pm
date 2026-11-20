package apistore

import (
	"testing"

	"github.com/xvantz/pm/internal/types"
)

// Before this change SaveBlocker could only create or resolve: editing the
// text silently did nothing on the daemon.
func TestRemote_EditBlockerPropagates(t *testing.T) {
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
		ID: "step", Title: "Step", Status: types.StepTodo,
		ProjectID: pd.Project.ID, CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveStep(step); err != nil {
		t.Fatalf("SaveStep error = %v", err)
	}
	blk := types.Blocker{
		ID: "no-hardware", Title: "No hardware", Reason: "old",
		Status: types.BlockerWaiting, ProjectID: pd.Project.ID, StepID: step.ID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveBlocker(blk); err != nil {
		t.Fatalf("SaveBlocker(new) error = %v", err)
	}

	blk.Title = "Hardware delayed"
	blk.Reason = "new reason"
	if err := st.SaveBlocker(blk); err != nil {
		t.Fatalf("SaveBlocker(edit) error = %v", err)
	}

	steps, err := st.GetSteps(pd.Project.ID)
	if err != nil {
		t.Fatalf("GetSteps error = %v", err)
	}
	if len(steps) != 1 || len(steps[0].Blockers) != 1 {
		t.Fatalf("unexpected shape: %+v", steps)
	}
	got := steps[0].Blockers[0]
	if got.Title != "Hardware delayed" {
		t.Errorf("title = %q, want %q", got.Title, "Hardware delayed")
	}
	if got.Reason != "new reason" {
		t.Errorf("reason = %q, want %q", got.Reason, "new reason")
	}
	if got.ID != "no-hardware" {
		t.Errorf("id moved on edit: %q", got.ID)
	}
}
