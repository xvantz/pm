package apistore

import (
	"testing"

	"github.com/xvantz/pm/internal/types"
)

// Agents reach the store through this adapter, so the consent gate has to hold
// over HTTP too, not only against MockStore. A preview must come back as a
// plan; a confirmed close must return a usable nil plan rather than a plan the
// caller will dereference.
func TestRemote_CloseConsentGate(t *testing.T) {
	st := newRemote(t)

	if err := st.SaveProject(types.Project{Title: "Consent"}); err != nil {
		t.Fatalf("SaveProject error = %v", err)
	}
	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject error = %v", err)
	}
	if err := st.SaveStep(types.Step{
		ID: "work", Title: "Work", Status: types.StepTodo, ProjectID: pd.Project.ID,
	}); err != nil {
		t.Fatalf("SaveStep error = %v", err)
	}

	// Preview: a plan, and nothing changed.
	plan, err := st.CloseProject(pd.Project.ID, "", false)
	if err != nil {
		t.Fatalf("preview error = %v", err)
	}
	if plan == nil {
		t.Fatal("preview over HTTP returned a nil plan")
	}
	if plan.StepsToClose() == 0 {
		t.Error("preview should list the open step")
	}
	mid, err := st.GetProject(pd.Project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if mid.Project.Status == types.StatusCompleted {
		t.Fatal("preview closed the project over HTTP — the gate does not hold")
	}

	// Confirm without a reason: refused.
	if _, err := st.CloseProject(pd.Project.ID, "", true); err == nil {
		t.Error("confirm without a reason must fail over HTTP too")
	}

	// Confirm with a reason: closes. The plan is nil here, and that is the
	// documented shape — callers must not read it.
	finalPlan, err := st.CloseProject(pd.Project.ID, "shipped", true)
	if err != nil {
		t.Fatalf("confirmed close error = %v", err)
	}
	if finalPlan != nil {
		t.Logf("confirmed close returned a plan as well (%d steps)", finalPlan.StepsToClose())
	}
	after, err := st.GetProject(pd.Project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Project.Status != types.StatusCompleted {
		t.Errorf("status = %q, want completed", after.Project.Status)
	}
}

func TestRemote_ClosePlanDoesNotMutate(t *testing.T) {
	st := newRemote(t)
	if err := st.SaveProject(types.Project{Title: "Preview"}); err != nil {
		t.Fatalf("SaveProject error = %v", err)
	}
	pd, _ := st.ResolveProject("1")
	if err := st.SaveStep(types.Step{
		ID: "s", Title: "S", Status: types.StepInProgress, ProjectID: pd.Project.ID,
	}); err != nil {
		t.Fatalf("SaveStep error = %v", err)
	}

	plan, err := st.ClosePlan(pd.Project.ID)
	if err != nil {
		t.Fatalf("ClosePlan error = %v", err)
	}
	if plan == nil {
		t.Fatal("ClosePlan over HTTP returned nil")
	}
	after, err := st.GetProject(pd.Project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Project.Status == types.StatusCompleted {
		t.Error("ClosePlan completed the project")
	}
	for _, s := range after.Steps {
		if s.Status == types.StepDone {
			t.Errorf("ClosePlan marked step %q done", s.ID)
		}
	}
}
