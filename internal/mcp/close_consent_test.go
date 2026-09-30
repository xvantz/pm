package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
)

// The consent gate as an agent meets it: the first call must hand back
// something to show a human and change nothing on disk.
func TestCloseProject_FirstCallPlansNotCloses(t *testing.T) {
	st := store.NewMockStore()
	pd, err := st.ResolveProject("2")
	if err != nil {
		t.Fatalf("ResolveProject error = %v", err)
	}

	out, err := handleCloseProject(st, context.Background(), json.RawMessage(`{"project_id":"2"}`))
	if err != nil {
		t.Fatalf("close_project error = %v", err)
	}
	if !strings.Contains(out, "NOT closed") {
		t.Errorf("the plan must state the project is not closed yet, got:\n%s", out)
	}
	if !strings.Contains(out, "confirm: true") {
		t.Errorf("the plan must tell the agent how to proceed, got:\n%s", out)
	}
	if !strings.Contains(out, "reason") {
		t.Errorf("the plan must mention the required reason, got:\n%s", out)
	}

	after, err := st.GetProject(pd.Project.ID)
	if err != nil {
		t.Fatalf("GetProject error = %v", err)
	}
	if after.Project.Status == "completed" {
		t.Error("the first call closed the project — the gate is not working")
	}
}

func TestCloseProject_ConfirmWithoutReasonFails(t *testing.T) {
	st := store.NewMockStore()
	pd, _ := st.ResolveProject("2")

	if _, err := handleCloseProject(st, context.Background(),
		json.RawMessage(`{"project_id":"2","confirm":true}`)); err == nil {
		t.Fatal("confirm without a reason must fail")
	} else if !strings.Contains(err.Error(), "reason") {
		t.Errorf("the error must name the reason, got: %v", err)
	}

	after, _ := st.GetProject(pd.Project.ID)
	if after.Project.Status == "completed" {
		t.Error("a refused close must not complete the project")
	}
}

func TestCloseProject_ConfirmWithReasonCloses(t *testing.T) {
	st := store.NewMockStore()
	pd, _ := st.ResolveProject("2")

	out, err := handleCloseProject(st, context.Background(),
		json.RawMessage(`{"project_id":"2","confirm":true,"reason":"shipped"}`))
	if err != nil {
		t.Fatalf("confirmed close error = %v", err)
	}
	if !strings.Contains(out, "closed") {
		t.Errorf("output should confirm the close, got: %s", out)
	}
	if !strings.Contains(out, "shipped") {
		t.Errorf("output should carry the reason, got: %s", out)
	}

	after, _ := st.GetProject(pd.Project.ID)
	if after.Project.Status != "completed" {
		t.Errorf("status = %q, want completed", after.Project.Status)
	}
}

// A blocker on a closing step is not settled by closing. The plan says so, so a
// human cannot approve believing it was.
func TestCloseProject_PlanStatesBlockersSurvive(t *testing.T) {
	st := store.NewMockStore()
	out, err := handleCloseProject(st, context.Background(), json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("close_project error = %v", err)
	}
	if !strings.Contains(out, "blocker") {
		t.Fatalf("project 1 has an unresolved blocker, the plan must mention it, got:\n%s", out)
	}
	if !strings.Contains(out, "does NOT resolve") {
		t.Errorf("the plan must say blockers are not resolved by closing, got:\n%s", out)
	}
}

func TestCloseProject_AlreadyCompleted(t *testing.T) {
	st := store.NewMockStore()
	pd, _ := st.ResolveProject("2")
	if _, err := handleCloseProject(st, context.Background(),
		json.RawMessage(`{"project_id":"2","confirm":true,"reason":"first"}`)); err != nil {
		t.Fatalf("first close error = %v", err)
	}
	if _, err := handleCloseProject(st, context.Background(),
		json.RawMessage(`{"project_id":"2","confirm":true,"reason":"again"}`)); err == nil {
		t.Error("closing a completed project must fail")
	}
	_ = pd
}
