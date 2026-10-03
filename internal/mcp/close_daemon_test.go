package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/api"
	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// The 2026-10-03 production SIGSEGV: confirm over the daemon answered with a
// nil plan while the mock answered with one, and handleCloseProject
// dereferenced it unconditionally. This test walks the daemon path, where the
// mock's lie cannot hide the divergence.
func TestCloseProjectConfirmViaDaemon(t *testing.T) {
	srv := api.New(store.NewFileStore(t.TempDir()), "tok")
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	st, ok := apistore.NewFromEnvVars(ts.URL, "tok")
	if !ok {
		t.Fatal("NewFromEnvVars refused test addr")
	}

	if err := st.SaveProject(types.Project{Title: "Daemon Close Probe"}); err != nil {
		t.Fatal(err)
	}
	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatal(err)
	}
	now := types.NowTimestamp()
	if err := st.SaveStep(types.Step{ID: "probe", Title: "probe", Status: types.StepTodo,
		ProjectID: pd.Project.ID, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	out, err := handleCloseProject(st, context.Background(),
		json.RawMessage(`{"project_id":"1","confirm":true,"reason":"probe done"}`))
	if err != nil {
		t.Fatalf("daemon confirm error = %v", err)
	}
	if !strings.Contains(out, "moved to done") {
		t.Errorf("confirm over daemon must carry the moved count, got %q", out)
	}

	after, err := st.ResolveProject("1")
	if err != nil {
		t.Fatal(err)
	}
	if after.Project.Status != types.StatusCompleted {
		t.Errorf("status = %q, want completed", after.Project.Status)
	}
}

// nilPlanStore mimics the old daemon contract (confirm answers without a
// plan). The handler must degrade to a count-less message, never panic: the
// process must outlive any single store answer.
type nilPlanStore struct {
	store.Store
}

func (n nilPlanStore) CloseProject(ref, reason string, confirm bool) (*types.ClosePlan, error) {
	if !confirm {
		return n.Store.CloseProject(ref, reason, false)
	}
	if _, err := n.Store.CloseProject(ref, reason, true); err != nil {
		return nil, err
	}
	return nil, nil
}

func TestCloseProjectConfirmSurvivesNilPlan(t *testing.T) {
	inner := store.NewMockStore()
	st := nilPlanStore{Store: inner}

	out, err := handleCloseProject(st, context.Background(),
		json.RawMessage(`{"project_id":"2","confirm":true,"reason":"shipped"}`))
	if err != nil {
		t.Fatalf("nil-plan confirm error = %v", err)
	}
	if !strings.Contains(out, "closed") || !strings.Contains(out, "shipped") {
		t.Errorf("must name the close and reason, got %q", out)
	}
}
