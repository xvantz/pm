package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// The whole point of the change, measured. A detail read of one project used
// to cost more than listing every project, which inverts the ratio: the
// overview must be the cheap answer and the detail the expensive one.
func TestBoundedReads_SummaryIsFarSmallerThanDetail(t *testing.T) {
	st := store.NewMockStore()

	summary, err := handleGetProject(st, context.Background(), json.RawMessage(`{"project_id":"2"}`))
	if err != nil {
		t.Fatalf("summary error = %v", err)
	}
	detail, err := handleGetProject(st, context.Background(), json.RawMessage(`{"project_id":"2","detail":true}`))
	if err != nil {
		t.Fatalf("detail error = %v", err)
	}

	ratio := float64(len(detail)) / float64(len(summary))
	t.Logf("project #2: summary=%d B, detail=%d B, ratio=%.1fx", len(summary), len(detail), ratio)
	if ratio < 3 {
		t.Errorf("summary must be at least 3x smaller than the detail dump, got %.1fx", ratio)
	}

	// And the summary must still stay small in absolute terms — a "small"
	// summary that grows with every step is just a slower full dump.
	if len(summary) > 500 {
		t.Errorf("summary is %d bytes, want under 500 for a 7-step project", len(summary))
	}
}

// The overview of everything must now cost less than one project's summary.
func TestBoundedReads_OverviewCheaperThanSingleDetail(t *testing.T) {
	st := store.NewMockStore()
	all, err := handleListProjects(st, context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list_projects error = %v", err)
	}
	detail, err := handleGetProject(st, context.Background(),
		json.RawMessage(`{"project_id":"1","detail":true}`))
	if err != nil {
		t.Fatalf("detail error = %v", err)
	}
	summary, err := handleGetProject(st, context.Background(), json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("summary error = %v", err)
	}
	t.Logf("all projects=%d B | one project summary=%d B | one project detail=%d B",
		len(all), len(summary), len(detail))
	if len(summary) >= len(detail) {
		t.Error("a project summary must be smaller than a project detail dump")
	}
}

// list_steps must be brief: no blockers, no artifacts.
func TestBoundedReads_ListStepsOmitsHeavyFields(t *testing.T) {
	st := store.NewMockStore()
	out, err := handleListSteps(st, context.Background(), json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("list_steps error = %v", err)
	}

	var payload struct {
		Count int `json:"count"`
		Steps []struct {
			ID        string            `json:"id"`
			Title     string            `json:"title"`
			Status    string            `json:"status"`
			UpdatedAt string            `json:"updated_at"`
			Blockers  []json.RawMessage `json:"blockers"`
			Artifacts []json.RawMessage `json:"artifacts"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	if payload.Count != len(payload.Steps) {
		t.Errorf("count = %d but %d steps returned", payload.Count, len(payload.Steps))
	}
	for _, s := range payload.Steps {
		if len(s.Blockers) != 0 {
			t.Errorf("step %q carries %d blockers; the brief listing must not", s.ID, len(s.Blockers))
		}
		if len(s.Artifacts) != 0 {
			t.Errorf("step %q carries %d artifacts; the brief listing must not", s.ID, len(s.Artifacts))
		}
	}
}

// The third level: one step, in full, without the project around it.
func TestBoundedReads_GetStepReturnsOneStepInFull(t *testing.T) {
	st := store.NewMockStore()
	out, err := handleGetStep(st, context.Background(),
		json.RawMessage(`{"project_id":"1","step_id":"configure-dns"}`))
	if err != nil {
		t.Fatalf("get_step error = %v", err)
	}
	var step types.Step
	if err := json.Unmarshal([]byte(out), &step); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	if step.ID != "configure-dns" {
		t.Errorf("step id = %q", step.ID)
	}
	if len(step.Blockers) == 0 {
		t.Error("get_step must carry the step's blockers — that is what it is for")
	}
	t.Logf("get_step(%q) = %d B", step.ID, len(out))
}

// A step id from another project must fail loudly rather than return nothing.
func TestBoundedReads_GetStepRejectsCrossProject(t *testing.T) {
	st := store.NewMockStore()
	// setup-caddy belongs to project 1, ask for it inside project 2.
	if _, err := handleGetStep(st, context.Background(),
		json.RawMessage(`{"project_id":"2","step_id":"setup-caddy"}`)); err == nil {
		t.Fatal("a step from another project must not resolve")
	} else if !strings.Contains(err.Error(), "setup-caddy") {
		t.Errorf("the error must name the step, got: %v", err)
	}

	if _, err := handleGetStep(st, context.Background(),
		json.RawMessage(`{"project_id":"1","step_id":"no-such-step"}`)); err == nil {
		t.Error("a missing step must be an error")
	}
	if _, err := handleGetStep(st, context.Background(),
		json.RawMessage(`{"project_id":"1"}`)); err == nil {
		t.Error("a missing step_id argument must be an error")
	}
}

// get_step must cost a fraction of the project detail it replaces.
func TestBoundedReads_GetStepBecheaperThanProjectDetail(t *testing.T) {
	st := store.NewMockStore()
	step, err := handleGetStep(st, context.Background(),
		json.RawMessage(`{"project_id":"1","step_id":"configure-dns"}`))
	if err != nil {
		t.Fatalf("get_step error = %v", err)
	}
	detail, err := handleGetProject(st, context.Background(),
		json.RawMessage(`{"project_id":"1","detail":true}`))
	if err != nil {
		t.Fatalf("detail error = %v", err)
	}
	t.Logf("one step=%d B, whole project=%d B", len(step), len(detail))
	if len(step) >= len(detail) {
		t.Error("one step must be smaller than the whole project dump")
	}
}

// The summary carries the hint that tells an agent where to go next, so a
// single call is enough to know what to do.
func TestBoundedReads_SummaryCarriesActionableHint(t *testing.T) {
	st := store.NewMockStore()
	cases := []struct {
		project  string
		wantWord string
	}{
		{"1", "list_blockers"}, // seeded unresolved blocker
		{"2", "list_steps"},    // open steps, no blockers
	}
	for _, c := range cases {
		args, err := json.Marshal(map[string]any{"project_id": c.project})
		if err != nil {
			t.Fatal(err)
		}
		out, err := handleGetProject(st, context.Background(), args)
		if err != nil {
			t.Fatalf("project %s error = %v", c.project, err)
		}
		var s jsonProjectSummary
		if err := json.Unmarshal([]byte(out), &s); err != nil {
			t.Fatalf("decode error = %v", err)
		}
		if !strings.Contains(s.Hint, c.wantWord) {
			t.Errorf("project %s hint = %q, want it to point at %s", c.project, s.Hint, c.wantWord)
		}
		if s.StepsTotal == 0 {
			t.Errorf("project %s summary reports no steps", c.project)
		}
	}
}

// Every step is accounted for in the counts: a summary must not hide steps.
func TestBoundedReads_SummaryCountsEveryStep(t *testing.T) {
	st := store.NewMockStore()
	projects, _ := st.ListProjects()
	for _, p := range projects {
		pd, err := st.GetProject(p.ID)
		if err != nil {
			continue
		}
		out, err := handleGetProject(st, context.Background(),
			json.RawMessage(`{"project_id":"`+p.ID+`"}`))
		if err != nil {
			t.Fatalf("project %s error = %v", p.ID, err)
		}
		var s jsonProjectSummary
		if err := json.Unmarshal([]byte(out), &s); err != nil {
			t.Fatal(err)
		}
		if s.StepsTotal != len(pd.Steps) {
			t.Errorf("project #%d: steps_total = %d, want %d", p.Number, s.StepsTotal, len(pd.Steps))
		}
		sum := 0
		for _, n := range s.StepsByStatus {
			sum += n
		}
		if sum != len(pd.Steps) {
			t.Errorf("project #%d: steps_by_status sums to %d, want %d", p.Number, sum, len(pd.Steps))
		}
		wantDone := 0
		for _, st := range pd.Steps {
			if st.Status == types.StepDone {
				wantDone++
			}
		}
		if s.StepsDone != wantDone {
			t.Errorf("project #%d: steps_done = %d, want %d", p.Number, s.StepsDone, wantDone)
		}
	}
}
