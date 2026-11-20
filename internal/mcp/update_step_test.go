package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/slug"
	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

func seedStep(t *testing.T, st store.Store, projectID, title string) types.Step {
	t.Helper()
	now := types.NowTimestamp()
	stp := types.Step{
		ID: slug.Of(title), Title: title, Status: types.StepTodo,
		ProjectID: projectID, CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveStep(stp); err != nil {
		t.Fatalf("seed step %q: %v", title, err)
	}
	return stp
}

func callUpdateStep(t *testing.T, st store.Store, args string) (string, error) {
	t.Helper()
	return handleUpdateStep(st, context.Background(), json.RawMessage(args))
}

func TestUpdateStep_RenameKeepsID(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	step := seedStep(t, st, p.ID, "Old Step")

	out, err := callUpdateStep(t, st, `{"project_id":"1","step_id":"old-step","title":"New Step"}`)
	if err != nil {
		t.Fatalf("update_step error: %v", err)
	}
	if !strings.Contains(out, "New Step") {
		t.Errorf("output does not report the new title: %s", out)
	}

	steps, err := st.GetSteps(p.ID)
	if err != nil {
		t.Fatalf("read steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(steps))
	}
	if steps[0].Title != "New Step" {
		t.Errorf("title = %q, want New Step", steps[0].Title)
	}
	if steps[0].ID != step.ID {
		t.Errorf("id moved: %q -> %q, want stable", step.ID, steps[0].ID)
	}
}

func TestUpdateStep_EmptyTitleRefused(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	seedStep(t, st, p.ID, "Keep Step")

	_, err := callUpdateStep(t, st, `{"project_id":"1","step_id":"keep-step","title":" "}`)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("error = %v, want a title complaint", err)
	}
	steps, _ := st.GetSteps(p.ID)
	if steps[0].Title != "Keep Step" {
		t.Errorf("title = %q after refused rename", steps[0].Title)
	}
}

func TestUpdateStep_UnknownStep(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "P", "")

	_, err := callUpdateStep(t, st, `{"project_id":"1","step_id":"nope","title":"X"}`)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want not found", err)
	}
}

func TestUpdateStep_DuplicateSlugRefused(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	seedStep(t, st, p.ID, "First Step")
	seedStep(t, st, p.ID, "Second Step")

	_, err := callUpdateStep(t, st, `{"project_id":"1","step_id":"first-step","title":"Second Step"}`)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want a duplicate complaint", err)
	}
}
