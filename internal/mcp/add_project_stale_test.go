package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// trashGapStore simulates the remote daemon after trash deletes: the live
// max is N but the server-side counter is ahead (N+k). NextNumber stays
// advisory (live max+1); SaveProject assigns the server number; GetProject
// returns the server number. Before the fix handleAddProject printed the
// advisory number, so the confirmation lied and add_step 404'd.
type trashGapStore struct {
	store.Store
	serverNext int
}

func (s *trashGapStore) NextNumber() (int, error) {
	projects, err := s.Store.ListProjects()
	if err != nil {
		return 0, err
	}
	max := 0
	for _, p := range projects {
		if p.Number > max {
			max = p.Number
		}
	}
	return max + 1, nil
}

func (s *trashGapStore) AdvanceNextNumber() error { return nil }

func (s *trashGapStore) SaveProject(p types.Project) error {
	if _, err := s.Store.GetProject(p.ID); err == nil {
		return s.Store.SaveProject(p)
	}
	p.Number = s.serverNext
	s.serverNext++
	return s.Store.SaveProject(p)
}

func TestHandleAddProject_TrashGapPrintsServerNumber(t *testing.T) {
	t.Parallel()
	inner := store.NewMockStore()
	// Mock seeds #1-#6, live max is 6. Simulate trash having consumed
	// numbers up to 13, so the daemon counter sits at 14.
	st := &trashGapStore{Store: inner, serverNext: 14}

	out, err := handleAddProject(st, context.Background(), json.RawMessage(`{"title":"Trash Gap Probe"}`))
	if err != nil {
		t.Fatalf("handleAddProject error: %v", err)
	}
	if !strings.Contains(out, "Project #14") {
		t.Errorf("confirmation must print server number #14, got %q", out)
	}
	if strings.Contains(out, "Project #7") {
		t.Errorf("confirmation must not print stale advisory #7, got %q", out)
	}

	// The printed Next identifier must work on first try (no manual list).
	id := extractID(t, out)
	stepOut, err := handleAddStep(st, context.Background(), json.RawMessage(`{"project_id":`+`"`+id+`" ,"title":"first step"}`))
	if err != nil {
		t.Fatalf("add_step with printed id %q: %v", id, err)
	}
	if !strings.Contains(stepOut, "first-step") {
		t.Errorf("add_step output missing slug, got %q", stepOut)
	}

	// Negative check: trash the project, create another - number still matches.
	pd, err := st.GetProject(id)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	_ = pd
	out2, err := handleAddProject(st, context.Background(), json.RawMessage(`{"title":"Trash Gap Probe 2"}`))
	if err != nil {
		t.Fatalf("second handleAddProject error: %v", err)
	}
	if !strings.Contains(out2, "Project #15") {
		t.Errorf("second confirmation must print #15, got %q", out2)
	}
	id2 := extractID(t, out2)
	if _, err := handleAddStep(st, context.Background(), json.RawMessage(`{"project_id":`+`"`+id2+`" ,"title":"s2"}`)); err != nil {
		t.Errorf("add_step with second printed id: %v", err)
	}
}

func extractID(t *testing.T, out string) string {
	t.Helper()
	// Confirmation format: "Project #N \"title\" created.\nID: <uuid>\n\nNext: ..."
	idx := strings.Index(out, "ID: ")
	if idx < 0 {
		t.Fatalf("no ID: in %q", out)
	}
	rest := strings.TrimSpace(out[idx+len("ID: "):])
	id := strings.Fields(rest)[0]
	if id == "" {
		t.Fatalf("empty id in %q", out)
	}
	// Next line must carry the same UUID (never a stale number).
	if !strings.Contains(out, `project_id: "`+id+`"`) {
		t.Errorf("Next: must reference UUID %q, got %q", id, out)
	}
	return id
}
