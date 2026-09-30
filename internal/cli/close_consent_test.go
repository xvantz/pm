package cli

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/xvantz/pm/internal/types"
)

// makeLiveProject creates a project with one open step through the daemon the
// whole package already talks to.
func makeLiveProject(t *testing.T, title string) types.Project {
	t.Helper()
	st, err := openStore()
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	uid, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	now := types.NowTimestamp()
	p := types.Project{
		ID: uid.String(), Title: title, Status: types.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveProject(p); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}
	saved, err := st.ResolveProject(p.ID)
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if err := st.SaveStep(types.Step{
		ID:        "work",
		Title:     "Work",
		Status:    types.StepTodo,
		ProjectID: p.ID,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveStep() error = %v", err)
	}
	return saved.Project
}

// A reason is mandatory: it is the only durable trace of why a project was
// closed, and the old default ("bulk close") recorded nothing.
func TestProjectClose_ReasonIsRequired(t *testing.T) {
	p := makeLiveProject(t, "Needs reason")

	err := cmdProjectClose([]string{p.ID})
	if err == nil {
		t.Fatal("close without a reason must fail")
	}
	if !strings.Contains(err.Error(), "reason") {
		t.Errorf("the error must name the reason, got: %v", err)
	}

	st, _ := openStore()
	after, err := st.GetProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Project.Status == types.StatusCompleted {
		t.Error("a refused close must not complete the project")
	}
}

func TestProjectClose_WithReasonCloses(t *testing.T) {
	p := makeLiveProject(t, "Closable")

	if err := cmdProjectClose([]string{p.ID, "shipped", "in", "v1.2"}); err != nil {
		t.Fatalf("close error = %v", err)
	}

	st, _ := openStore()
	after, err := st.GetProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
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
		if d.ID == "closed" && strings.Contains(d.Reason, "shipped in v1.2") {
			found = true
		}
	}
	if !found {
		t.Errorf("the multi-word reason must reach the decision, got %+v", after.Decisions)
	}
}
