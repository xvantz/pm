package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
)

type listProjectsResp struct {
	Count          int               `json:"count"`
	Projects       []jsonProjectItem `json:"projects"`
	ActiveCount    int               `json:"active_count"`
	CompletedCount int               `json:"completed_count"`
	Total          int               `json:"total"`
	Hint           string            `json:"hint,omitempty"`
}

func decodeListProjects(t *testing.T, raw string) listProjectsResp {
	t.Helper()
	var r listProjectsResp
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("decode list_projects: %v: %s", err, raw)
	}
	return r
}

func closeOneProject(t *testing.T, st *store.MockStore) {
	t.Helper()
	if _, err := st.CloseProject("1", "test close", true); err != nil {
		t.Fatalf("CloseProject: %v", err)
	}
}

// Default filters but counts: the cheap answer must COUNT what it hid.
func TestListProjects_Filter_DefaultActiveCountsAll(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	closeOneProject(t, st)

	raw, err := handleListProjects(st, context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list_projects: %v", err)
	}
	r := decodeListProjects(t, raw)

	if r.Total == 0 || r.CompletedCount != 1 {
		t.Errorf("want total>0 with 1 completed, got total=%d completed=%d: %s", r.Total, r.CompletedCount, raw)
	}
	if r.ActiveCount+r.CompletedCount != r.Total {
		t.Errorf("counts must sum to total: active=%d completed=%d total=%d", r.ActiveCount, r.CompletedCount, r.Total)
	}
	if r.Count != len(r.Projects) {
		t.Errorf("count=%d but projects has %d entries", r.Count, len(r.Projects))
	}
	if r.Count != r.ActiveCount {
		t.Errorf("default view must carry active only: count=%d active=%d", r.Count, r.ActiveCount)
	}
	for _, p := range r.Projects {
		if p.Status == "completed" {
			t.Errorf("default view leaked completed project #%d", p.Number)
		}
	}
	if !strings.Contains(raw, `"hint"`) {
		t.Errorf("default view hiding completed must carry a hint: %s", raw)
	}
}

func TestListProjects_Filter_AllAndCompleted(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	closeOneProject(t, st)

	allRaw, err := handleListProjects(st, context.Background(), json.RawMessage(`{"status":"all"}`))
	if err != nil {
		t.Fatalf("status=all: %v", err)
	}
	all := decodeListProjects(t, allRaw)
	if all.Count != all.Total || all.Count != all.ActiveCount+all.CompletedCount {
		t.Errorf("all must return everything: count=%d total=%d active=%d completed=%d",
			all.Count, all.Total, all.ActiveCount, all.CompletedCount)
	}

	compRaw, err := handleListProjects(st, context.Background(), json.RawMessage(`{"status":"completed"}`))
	if err != nil {
		t.Fatalf("status=completed: %v", err)
	}
	comp := decodeListProjects(t, compRaw)
	if comp.Count != comp.CompletedCount || comp.Count != 1 {
		t.Errorf("completed view must carry only closed: count=%d completed=%d", comp.Count, comp.CompletedCount)
	}
	for _, p := range comp.Projects {
		if p.Status != "completed" {
			t.Errorf("completed view leaked status %q (#%d)", p.Status, p.Number)
		}
	}
}

func TestListProjects_Filter_UnknownStatus(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	_, err := handleListProjects(st, context.Background(), json.RawMessage(`{"status":"deleted"}`))
	if err == nil {
		t.Fatal("expected error for unknown status")
	}
	if !strings.Contains(err.Error(), "active|completed|all") {
		t.Errorf("error must name accepted values, got: %v", err)
	}
}
