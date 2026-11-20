package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/xvantz/pm/internal/types"
)

// makeProject creates a project through the API and returns it.
func makeProject(t *testing.T, srv *Server, title string) types.Project {
	t.Helper()
	w := doReq(t, srv, "POST", "/api/projects", map[string]any{"title": title}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project: code %d body %s", w.Code, w.Body.String())
	}
	var p types.Project
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	return p
}

func TestProjectPatch_TitleRenamesOnly(t *testing.T) {
	srv := newTestServer(t)
	p := makeProject(t, srv, "Old Title")
	w := doReq(t, srv, "PATCH", "/api/projects/1", map[string]any{"title": "New Title"}, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch title: code %d body %s", w.Code, w.Body.String())
	}
	var got types.Project
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Title != "New Title" {
		t.Errorf("title = %q, want %q", got.Title, "New Title")
	}
	if got.Goal != p.Goal || got.Status != p.Status {
		t.Errorf("other fields changed: goal %q->%q, status %q->%q", p.Goal, got.Goal, p.Status, got.Status)
	}
}

func TestProjectPatch_EmptyTitleRefused(t *testing.T) {
	srv := newTestServer(t)
	makeProject(t, srv, "Keep Me")
	w := doReq(t, srv, "PATCH", "/api/projects/1", map[string]any{"title": "   "}, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty title: code %d, want 400 (body %s)", w.Code, w.Body.String())
	}
	got := doReq(t, srv, "GET", "/api/projects/1", nil, testToken)
	if !json.Valid(got.Body.Bytes()) {
		t.Fatalf("get project: %s", got.Body.String())
	}
	var pd types.ProjectData
	if err := json.Unmarshal(got.Body.Bytes(), &pd); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pd.Project.Title != "Keep Me" {
		t.Errorf("title = %q after refused patch, want %q", pd.Project.Title, "Keep Me")
	}
}

func TestStepPatch_RenamesKeepsID(t *testing.T) {
	srv := newTestServer(t)
	makeProject(t, srv, "P")
	w := doReq(t, srv, "POST", "/api/projects/1/steps", map[string]any{"title": "Old Step"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("add step: code %d body %s", w.Code, w.Body.String())
	}
	var created types.Step
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode step: %v", err)
	}

	w = doReq(t, srv, "PATCH", "/api/projects/1/steps/"+created.ID, map[string]any{"title": "New Step Name"}, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch step: code %d body %s", w.Code, w.Body.String())
	}
	var got types.Step
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Title != "New Step Name" {
		t.Errorf("title = %q, want %q", got.Title, "New Step Name")
	}
	if got.ID != created.ID {
		t.Errorf("id moved with the title: %q -> %q, want stable %q", created.ID, got.ID, created.ID)
	}

	list := doReq(t, srv, "GET", "/api/projects/1/steps", nil, testToken)
	var steps []types.Step
	if err := json.Unmarshal(list.Body.Bytes(), &steps); err != nil {
		t.Fatalf("decode steps: %v", err)
	}
	if len(steps) != 1 || steps[0].Title != "New Step Name" || steps[0].ID != created.ID {
		t.Errorf("stored steps = %+v, want one renamed step with stable id", steps)
	}
}

func TestStepPatch_EmptyTitleRefused(t *testing.T) {
	srv := newTestServer(t)
	makeProject(t, srv, "P")
	w := doReq(t, srv, "POST", "/api/projects/1/steps", map[string]any{"title": "Keep Step"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("add step: %d %s", w.Code, w.Body.String())
	}
	var created types.Step
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	w = doReq(t, srv, "PATCH", "/api/projects/1/steps/"+created.ID, map[string]any{"title": ""}, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty title: code %d, want 400", w.Code)
	}
}

func TestStepPatch_UnknownStepNotFound(t *testing.T) {
	srv := newTestServer(t)
	makeProject(t, srv, "P")
	w := doReq(t, srv, "PATCH", "/api/projects/1/steps/nope", map[string]any{"title": "X"}, testToken)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown step: code %d, want 404", w.Code)
	}
}

func TestStepPatch_DuplicateSlugConflict(t *testing.T) {
	srv := newTestServer(t)
	makeProject(t, srv, "P")
	for _, title := range []string{"First Step", "Second Step"} {
		w := doReq(t, srv, "POST", "/api/projects/1/steps", map[string]any{"title": title}, testToken)
		if w.Code != http.StatusCreated {
			t.Fatalf("add step %q: %d %s", title, w.Code, w.Body.String())
		}
	}
	w := doReq(t, srv, "PATCH", "/api/projects/1/steps/first-step", map[string]any{"title": "Second Step"}, testToken)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate rename: code %d, want 409 (body %s)", w.Code, w.Body.String())
	}
}
