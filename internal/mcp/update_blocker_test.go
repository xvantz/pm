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

func seedBlocker(t *testing.T, st store.Store, step types.Step, title, reason string) types.Blocker {
	t.Helper()
	now := types.NowTimestamp()
	b := types.Blocker{
		ID: slug.Of(title), Title: title, Reason: reason,
		Status: types.BlockerWaiting, ProjectID: step.ProjectID, StepID: step.ID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveBlocker(b); err != nil {
		t.Fatalf("seed blocker %q: %v", title, err)
	}
	return b
}

func callUpdateBlocker(t *testing.T, st store.Store, args string) (string, error) {
	t.Helper()
	return handleUpdateBlocker(st, context.Background(), json.RawMessage(args))
}

func TestUpdateBlocker_RenameKeepsID(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	step := seedStep(t, st, p.ID, "Step")
	blk := seedBlocker(t, st, step, "No hardware", "waiting")

	if _, err := callUpdateBlocker(t, st, `{"project_id":"1","step_id":"step","blocker_id":"no-hardware","title":"Hardware delayed"}`); err != nil {
		t.Fatalf("update_blocker error: %v", err)
	}

	steps, err := st.GetSteps(p.ID)
	if err != nil {
		t.Fatalf("read steps: %v", err)
	}
	var got *types.Blocker
	for i := range steps[0].Blockers {
		if steps[0].Blockers[i].ID == blk.ID {
			got = &steps[0].Blockers[i]
		}
	}
	if got == nil {
		t.Fatalf("blocker %q disappeared", blk.ID)
	}
	if got.Title != "Hardware delayed" {
		t.Errorf("title = %q, want Hardware delayed", got.Title)
	}
	if got.Reason != "waiting" {
		t.Errorf("reason = %q, want untouched", got.Reason)
	}
}

func TestUpdateBlocker_ReasonOnly(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	step := seedStep(t, st, p.ID, "Step")
	seedBlocker(t, st, step, "Blocker", "old")

	if _, err := callUpdateBlocker(t, st, `{"project_id":"1","step_id":"step","blocker_id":"blocker","reason":"new"}`); err != nil {
		t.Fatalf("update_blocker error: %v", err)
	}
	steps, _ := st.GetSteps(p.ID)
	if steps[0].Blockers[0].Reason != "new" || steps[0].Blockers[0].Title != "Blocker" {
		t.Errorf("got %+v, want reason new and title Blocker", steps[0].Blockers[0])
	}
}

func TestUpdateBlocker_NoFieldsRefused(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	step := seedStep(t, st, p.ID, "Step")
	seedBlocker(t, st, step, "Blocker", "")

	_, err := callUpdateBlocker(t, st, `{"project_id":"1","step_id":"step","blocker_id":"blocker"}`)
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("error = %v, want 'nothing to update'", err)
	}
}

func TestUpdateBlocker_UnknownBlockers(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	seedStep(t, st, p.ID, "Step")

	if _, err := callUpdateBlocker(t, st, `{"project_id":"1","step_id":"step","blocker_id":"nope","title":"X"}`); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown blocker: error = %v, want not found", err)
	}
	if _, err := callUpdateBlocker(t, st, `{"project_id":"1","step_id":"nope","blocker_id":"nope","title":"X"}`); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown step: error = %v, want not found", err)
	}
}

func TestUpdateBlocker_DuplicateSlugRefused(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	p := seedProject(t, st, "P", "")
	step := seedStep(t, st, p.ID, "Step")
	seedBlocker(t, st, step, "First blocker", "")
	seedBlocker(t, st, step, "Second blocker", "")

	_, err := callUpdateBlocker(t, st, `{"project_id":"1","step_id":"step","blocker_id":"first-blocker","title":"Second blocker"}`)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want duplicate complaint", err)
	}
}
