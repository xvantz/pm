package apistore

import (
	"net/http/httptest"
	"testing"

	"github.com/xvantz/pm/internal/api"
	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

func newRemote(t *testing.T) store.Store {
	t.Helper()
	srv := api.New(store.NewFileStore(t.TempDir()), "tok")
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	st, ok := NewFromEnvVars(ts.URL, "tok")
	if !ok {
		t.Fatal("NewFromEnvVars refused non-empty addr")
	}
	return st
}

func TestRemoteProjectLifecycle(t *testing.T) {
	st := newRemote(t)

	if _, ok := NewFromEnvVars("", "tok"); ok {
		t.Error("empty addr must report false")
	}

	n, err := st.NextNumber()
	if err != nil || n != 1 {
		t.Fatalf("NextNumber = %d, %v; want 1", n, err)
	}
	if err := st.AdvanceNextNumber(); err != nil {
		t.Fatalf("AdvanceNextNumber (no-op) error = %v", err)
	}

	if err := st.SaveProject(types.Project{Title: "Remote"}); err != nil {
		t.Fatalf("SaveProject(new) error = %v", err)
	}
	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject(1) error = %v", err)
	}
	if pd.Project.Title != "Remote" || pd.Project.Number != 1 {
		t.Fatalf("project = %+v", pd.Project)
	}

	pd.Project.Goal = "g"
	pd.Project.Status = types.StatusActive
	if err := st.SaveProject(pd.Project); err != nil {
		t.Fatalf("SaveProject(existing) error = %v", err)
	}
	pd2, _ := st.ResolveProject("1")
	if pd2.Project.Goal != "g" || pd2.Project.Status != types.StatusActive {
		t.Fatalf("patched = %+v", pd2.Project)
	}
}

func TestRemoteStepLifecycle(t *testing.T) {
	st := newRemote(t)
	if err := st.SaveProject(types.Project{Title: "S"}); err != nil {
		t.Fatal(err)
	}
	pd, _ := st.ResolveProject("1")

	if err := st.SaveStep(types.Step{ID: "do", Title: "Do", Status: types.StepTodo, ProjectID: pd.Project.ID}); err != nil {
		t.Fatalf("SaveStep(new) error = %v", err)
	}
	steps, _ := st.GetSteps(pd.Project.ID)
	if len(steps) != 1 || steps[0].Status != types.StepTodo {
		t.Fatalf("steps = %+v", steps)
	}

	// start → review → done through delta mapping
	for _, want := range []types.StepStatus{types.StepInProgress, types.StepReview, types.StepDone} {
		pd, _ := st.ResolveProject("1")
		var cur types.Step
		for _, s := range pd.Steps {
			if s.ID == "do" {
				cur = s
			}
		}
		cur.Status = want
		if err := st.SaveStep(cur); err != nil {
			t.Fatalf("SaveStep(%s) error = %v", want, err)
		}
	}
	pd, _ = st.ResolveProject("1")
	for _, s := range pd.Steps {
		if s.ID == "do" && s.Status != types.StepDone {
			t.Fatalf("final status = %s", s.Status)
		}
	}

	// illegal delta surfaces an error, not a silent write
	pd, _ = st.ResolveProject("1")
	for _, s := range pd.Steps {
		if s.ID == "do" {
			s.Status = types.StepTodo
			if err := st.SaveStep(s); err == nil {
				t.Error("done→todo must fail")
			}
		}
	}
}

func TestRemoteBlockerFlow(t *testing.T) {
	st := newRemote(t)
	if err := st.SaveProject(types.Project{Title: "B"}); err != nil {
		t.Fatal(err)
	}
	pd, _ := st.ResolveProject("1")
	pid := pd.Project.ID

	if err := st.SaveStep(types.Step{ID: "w", Title: "W", Status: types.StepTodo, ProjectID: pid}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveBlocker(types.Blocker{ID: "b", Title: "B", Status: types.BlockerWaiting, ProjectID: pid, StepID: "w"}); err != nil {
		t.Fatalf("SaveBlocker(new) error = %v", err)
	}
	bl, _ := st.GetBlockers(pid)
	if len(bl) != 1 {
		t.Fatalf("blockers = %+v", bl)
	}

	// resolve via status flip, step unblocks server-side
	if err := st.SaveBlocker(types.Blocker{ID: "b", Title: "B", Status: types.BlockerResolved, ProjectID: pid, StepID: "w"}); err != nil {
		t.Fatalf("SaveBlocker(resolve) error = %v", err)
	}
	steps, _ := st.GetSteps(pid)
	for _, s := range steps {
		if s.ID == "w" && s.Status == types.StepBlocked {
			t.Error("step still blocked after resolve")
		}
	}

	if err := st.SaveDecision(types.Decision{ID: "d", Title: "D", ProjectID: pid}); err != nil {
		t.Fatalf("SaveDecision error = %v", err)
	}
	ds, _ := st.GetDecisions(pid)
	if len(ds) != 1 {
		t.Fatalf("decisions = %+v", ds)
	}
	// decisions immutable: re-save is a no-op
	if err := st.SaveDecision(types.Decision{ID: "d", Title: "D2", ProjectID: pid}); err != nil {
		t.Fatalf("SaveDecision(existing) error = %v", err)
	}

	if err := st.DeleteBlocker(pid, "w", "b"); err != nil {
		t.Fatalf("DeleteBlocker error = %v", err)
	}
	if err := st.DeleteDecision(pid, "d"); err != nil {
		t.Fatalf("DeleteDecision error = %v", err)
	}
	if err := st.DeleteStep(pid, "w"); err != nil {
		t.Fatalf("DeleteStep error = %v", err)
	}
}

func TestRemoteTrash(t *testing.T) {
	st := newRemote(t)
	if err := st.SaveProject(types.Project{Title: "T"}); err != nil {
		t.Fatal(err)
	}
	pd, _ := st.ResolveProject("1")
	if err := st.DeleteProject(pd.Project.ID); err != nil {
		t.Fatalf("DeleteProject error = %v", err)
	}
	names, err := st.TrashList()
	if err != nil || len(names) != 1 {
		t.Fatalf("TrashList = %v, %v", names, err)
	}
	if err := st.TrashRestore(names[0]); err != nil {
		t.Fatalf("TrashRestore error = %v", err)
	}
	if _, err := st.ResolveProject("1"); err != nil {
		t.Fatalf("ResolveProject after restore error = %v", err)
	}
	if err := st.DeleteProject(pd.Project.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.TrashClean(); err != nil {
		t.Fatalf("TrashClean error = %v", err)
	}
	names, _ = st.TrashList()
	if len(names) != 0 {
		t.Fatalf("trash not empty: %v", names)
	}
}
