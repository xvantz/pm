package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/xvantz/pm/internal/types"
)

// makeStepAndBlocker creates a project with one step and one blocker.
func makeStepAndBlocker(t *testing.T, srv *Server, title, reason string) (types.Step, types.Blocker) {
	t.Helper()
	makeProject(t, srv, "P")
	w := doReq(t, srv, "POST", "/api/projects/1/steps", map[string]any{"title": "Step"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("add step: %d %s", w.Code, w.Body.String())
	}
	var step types.Step
	if err := json.Unmarshal(w.Body.Bytes(), &step); err != nil {
		t.Fatalf("decode step: %v", err)
	}
	w = doReq(t, srv, "POST", "/api/projects/1/steps/"+step.ID+"/blockers",
		map[string]any{"title": title, "reason": reason}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("add blocker: %d %s", w.Code, w.Body.String())
	}
	var blk types.Blocker
	if err := json.Unmarshal(w.Body.Bytes(), &blk); err != nil {
		t.Fatalf("decode blocker: %v", err)
	}
	return step, blk
}

func TestBlockerPatch_RenameKeepsID(t *testing.T) {
	srv := newTestServer(t)
	step, blk := makeStepAndBlocker(t, srv, "No hardware", "waiting for delivery")

	w := doReq(t, srv, "PATCH", "/api/projects/1/steps/"+step.ID+"/blockers/"+blk.ID,
		map[string]any{"title": "Hardware delayed"}, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch blocker: %d %s", w.Code, w.Body.String())
	}
	var got types.Blocker
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Title != "Hardware delayed" {
		t.Errorf("title = %q, want %q", got.Title, "Hardware delayed")
	}
	if got.ID != blk.ID {
		t.Errorf("id moved on rename: %q -> %q", blk.ID, got.ID)
	}
	if got.Reason != "waiting for delivery" {
		t.Errorf("reason = %q, want it untouched", got.Reason)
	}
}

func TestBlockerPatch_ReasonOnly(t *testing.T) {
	srv := newTestServer(t)
	step, blk := makeStepAndBlocker(t, srv, "Blocker", "old reason")

	w := doReq(t, srv, "PATCH", "/api/projects/1/steps/"+step.ID+"/blockers/"+blk.ID,
		map[string]any{"reason": "new reason"}, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch reason: %d %s", w.Code, w.Body.String())
	}
	var got types.Blocker
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Reason != "new reason" || got.Title != "Blocker" {
		t.Errorf("got title %q reason %q, want Blocker / new reason", got.Title, got.Reason)
	}
}

func TestBlockerPatch_NoFieldsRefused(t *testing.T) {
	srv := newTestServer(t)
	step, blk := makeStepAndBlocker(t, srv, "Blocker", "")
	w := doReq(t, srv, "PATCH", "/api/projects/1/steps/"+step.ID+"/blockers/"+blk.ID, map[string]any{}, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty patch: %d, want 400", w.Code)
	}
}

func TestBlockerPatch_EmptyTitleRefused(t *testing.T) {
	srv := newTestServer(t)
	step, blk := makeStepAndBlocker(t, srv, "Blocker", "")
	w := doReq(t, srv, "PATCH", "/api/projects/1/steps/"+step.ID+"/blockers/"+blk.ID,
		map[string]any{"title": "  "}, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty title: %d, want 400", w.Code)
	}
}

func TestBlockerPatch_UnknownBlocker(t *testing.T) {
	srv := newTestServer(t)
	makeProject(t, srv, "P")
	w := doReq(t, srv, "POST", "/api/projects/1/steps", map[string]any{"title": "Step"}, testToken)
	var step types.Step
	if err := json.Unmarshal(w.Body.Bytes(), &step); err != nil {
		t.Fatalf("decode step: %v", err)
	}
	w = doReq(t, srv, "PATCH", "/api/projects/1/steps/"+step.ID+"/blockers/nope",
		map[string]any{"title": "X"}, testToken)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown blocker: %d, want 404", w.Code)
	}
}

func TestBlockerPatch_DuplicateSlugConflict(t *testing.T) {
	srv := newTestServer(t)
	step, first := makeStepAndBlocker(t, srv, "First blocker", "")
	w := doReq(t, srv, "POST", "/api/projects/1/steps/"+step.ID+"/blockers",
		map[string]any{"title": "Second blocker"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("add second blocker: %d %s", w.Code, w.Body.String())
	}
	w = doReq(t, srv, "PATCH", "/api/projects/1/steps/"+step.ID+"/blockers/"+first.ID,
		map[string]any{"title": "Second blocker"}, testToken)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate rename: %d, want 409", w.Code)
	}
}
