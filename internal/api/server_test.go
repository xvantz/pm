package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

const testToken = "test-token-123"

func newTestServer(t *testing.T) *Server {
	t.Helper()
	st := store.NewFileStore(t.TempDir())
	return New(st, testToken)
}

func doReq(t *testing.T, srv *Server, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func TestHealthzNoAuth(t *testing.T) {
	srv := newTestServer(t)
	w := doReq(t, srv, "GET", "/healthz", nil, "")
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", w.Code)
	}
	var got map[string]string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "ok" {
		t.Errorf("status = %q, want ok", got["status"])
	}
}

func TestAuthRequired(t *testing.T) {
	srv := newTestServer(t)
	for _, path := range []string{"/api/projects", "/api/briefing"} {
		w := doReq(t, srv, "GET", path, nil, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without token: code = %d, want 401", path, w.Code)
		}
		w = doReq(t, srv, "GET", path, nil, "wrong")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s wrong token: code = %d, want 401", path, w.Code)
		}
	}
}

func createProject(t *testing.T, srv *Server, title string) types.Project {
	t.Helper()
	w := doReq(t, srv, "POST", "/api/projects",
		map[string]string{"title": title}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create project: code = %d, body = %s", w.Code, w.Body.String())
	}
	var p types.Project
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.Number == 0 || p.ID == "" {
		t.Fatalf("project missing id/number: %+v", p)
	}
	return p
}

func TestProjectRoundTrip(t *testing.T) {
	srv := newTestServer(t)
	p := createProject(t, srv, "Daemon Test")

	w := doReq(t, srv, "GET", "/api/projects", nil, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list: code = %d", w.Code)
	}
	var list []types.Project
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Title != "Daemon Test" {
		t.Fatalf("list = %+v", list)
	}

	// Resolve by number.
	w = doReq(t, srv, "GET", fmt.Sprintf("/api/projects/%d", p.Number), nil, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get by number: code = %d", w.Code)
	}

	// Unknown ref -> 404.
	w = doReq(t, srv, "GET", "/api/projects/9999", nil, testToken)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown ref: code = %d, want 404", w.Code)
	}

	// Empty title -> 400.
	w = doReq(t, srv, "POST", "/api/projects",
		map[string]string{"title": "  "}, testToken)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty title: code = %d, want 400", w.Code)
	}
}

func TestStepLifecycle(t *testing.T) {
	srv := newTestServer(t)
	p := createProject(t, srv, "Lifecycle")
	base := fmt.Sprintf("/api/projects/%d/steps", p.Number)

	w := doReq(t, srv, "POST", base, map[string]string{"title": "Do thing"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create step: code = %d, body = %s", w.Code, w.Body.String())
	}
	var step types.Step
	if err := json.NewDecoder(w.Body).Decode(&step); err != nil {
		t.Fatal(err)
	}

	// Duplicate -> 409.
	w = doReq(t, srv, "POST", base, map[string]string{"title": "Do thing"}, testToken)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate step: code = %d, want 409", w.Code)
	}

	// done straight from todo -> 422 (review gate).
	w = doReq(t, srv, "POST", base+"/do-thing/done", nil, testToken)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("done from todo: code = %d, want 422", w.Code)
	}

	for _, action := range []string{"start", "review", "done"} {
		w = doReq(t, srv, "POST", base+"/do-thing/"+action, nil, testToken)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: code = %d, body = %s", action, w.Code, w.Body.String())
		}
	}
	var done types.Step
	if err := json.NewDecoder(w.Body).Decode(&done); err != nil {
		t.Fatal(err)
	}
	if done.Status != types.StepDone {
		t.Errorf("status = %q, want done", done.Status)
	}

	// Bad action -> 400.
	w = doReq(t, srv, "POST", base+"/do-thing/launch", nil, testToken)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad action: code = %d, want 400", w.Code)
	}
}

func TestBlockerAndDecision(t *testing.T) {
	srv := newTestServer(t)
	p := createProject(t, srv, "Blockers")
	base := fmt.Sprintf("/api/projects/%d", p.Number)

	w := doReq(t, srv, "POST", base+"/steps", map[string]string{"title": "Work"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create step: code = %d", w.Code)
	}

	w = doReq(t, srv, "POST", base+"/steps/work/blockers",
		map[string]string{"title": "No budget", "reason": "Q3"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create blocker: code = %d, body = %s", w.Code, w.Body.String())
	}

	// Blocked step cannot start (SaveBlocker invariant: step -> blocked).
	w = doReq(t, srv, "POST", base+"/steps/work/start", nil, testToken)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("start blocked step: code = %d, want 422", w.Code)
	}

	w = doReq(t, srv, "POST", base+"/steps/work/blockers/no-budget/resolve", nil, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("resolve: code = %d, body = %s", w.Code, w.Body.String())
	}

	// After resolve the step is back to todo: full lifecycle passes.
	for _, action := range []string{"start", "review", "done"} {
		w = doReq(t, srv, "POST", base+"/steps/work/"+action, nil, testToken)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: code = %d, body = %s", action, w.Code, w.Body.String())
		}
	}

	w = doReq(t, srv, "POST", base+"/decisions",
		map[string]string{"title": "Ship it", "reason": "why not"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create decision: code = %d, body = %s", w.Code, w.Body.String())
	}

	w = doReq(t, srv, "GET", base+"/decisions", nil, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list decisions: code = %d", w.Code)
	}
	var decisions []types.Decision
	if err := json.NewDecoder(w.Body).Decode(&decisions); err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 {
		t.Errorf("decisions = %d, want 1", len(decisions))
	}
}

func TestBriefing(t *testing.T) {
	srv := newTestServer(t)
	createProject(t, srv, "Briefed")
	w := doReq(t, srv, "GET", "/api/briefing", nil, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("briefing: code = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestProjectCreateID(t *testing.T) {
	srv := newTestServer(t)

	// Client-provided UUID is honored.
	w := doReq(t, srv, "POST", "/api/projects",
		map[string]string{"title": "Pinned", "id": "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create with id: code = %d, body = %s", w.Code, w.Body.String())
	}
	var p types.Project
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.ID != "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a" {
		t.Errorf("id = %q, want client-provided", p.ID)
	}

	// Same id twice -> 409.
	w = doReq(t, srv, "POST", "/api/projects",
		map[string]string{"title": "Dup", "id": "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a"}, testToken)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate id: code = %d, want 409", w.Code)
	}

	// Garbage id -> 400.
	w = doReq(t, srv, "POST", "/api/projects",
		map[string]string{"title": "Bad", "id": "not-a-uuid"}, testToken)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad id: code = %d, want 400", w.Code)
	}
}

func TestProjectClose(t *testing.T) {
	srv := newTestServer(t)
	p := createProject(t, srv, "Closer")
	base := "/api/projects"
	w := doReq(t, srv, "POST", base+"/1/steps", map[string]string{"title": "Work"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create step: code = %d", w.Code)
	}
	w = doReq(t, srv, "POST", base+"/1/close", map[string]string{"reason": "test done"}, testToken)
	if w.Code != http.StatusOK {
		t.Fatalf("close: code = %d, body = %s", w.Code, w.Body.String())
	}
	var closed types.Project
	if err := json.NewDecoder(w.Body).Decode(&closed); err != nil {
		t.Fatal(err)
	}
	if closed.Status != types.StatusCompleted {
		t.Errorf("status = %q, want completed", closed.Status)
	}
	_ = p
	w = doReq(t, srv, "POST", base+"/1/close", map[string]string{"reason": "again"}, testToken)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("double close: code = %d, want 422", w.Code)
	}
}
