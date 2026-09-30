package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
)

func TestBlockerRefs_ListStepsNamesBlockers(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	out, err := handleListSteps(st, context.Background(), json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("list_steps error = %v", err)
	}
	var payload struct {
		Count int `json:"count"`
		Steps []struct {
			ID         string   `json:"id"`
			BlockerIDs []string `json:"blocker_ids"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	found := false
	for _, s := range payload.Steps {
		if s.ID == "configure-dns" {
			found = true
			if len(s.BlockerIDs) != 1 || s.BlockerIDs[0] != "router" {
				t.Errorf("configure-dns blocker_ids = %v, want [router]", s.BlockerIDs)
			}
		}
		if s.ID == "test-dns" && len(s.BlockerIDs) != 0 {
			t.Errorf("test-dns blocker_ids = %v, want empty", s.BlockerIDs)
		}
	}
	if !found {
		t.Error("configure-dns missing from list_steps")
	}
}

func TestBlockerRefs_ListBlockersFiltersByStep(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	out, err := handleListBlockers(st, context.Background(),
		json.RawMessage(`{"project_id":"1","step_id":"configure-dns"}`))
	if err != nil {
		t.Fatalf("filtered list_blockers error = %v", err)
	}
	if !strings.Contains(out, `"step_id":"configure-dns"`) {
		t.Errorf("filtered output missing the step group: %s", out)
	}
	if !strings.Contains(out, `"id":"router"`) {
		t.Errorf("filtered output missing blocker router: %s", out)
	}

	full, err := handleListBlockers(st, context.Background(), json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("unfiltered list_blockers error = %v", err)
	}
	var filtered struct {
		Blockers []struct {
			StepID string `json:"step_id"`
		} `json:"blockers"`
	}
	if err := json.Unmarshal([]byte(out), &filtered); err != nil {
		t.Fatal(err)
	}
	if len(filtered.Blockers) != 1 || filtered.Blockers[0].StepID != "configure-dns" {
		t.Errorf("filtered output must carry exactly the configure-dns group: %s", out)
	}
	_ = full
}

func TestBlockerRefs_ListBlockersUnknownStepFails(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	_, err := handleListBlockers(st, context.Background(),
		json.RawMessage(`{"project_id":"1","step_id":"nope"}`))
	if err == nil {
		t.Fatal("unknown step_id must be an error")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error must name the step, got: %v", err)
	}
}

func TestBlockerRefs_IdsLeadToGetStep(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	list, err := handleListSteps(st, context.Background(), json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("list_steps error = %v", err)
	}
	var payload struct {
		Steps []struct {
			ID         string   `json:"id"`
			Status     string   `json:"status"`
			BlockerIDs []string `json:"blocker_ids"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(list), &payload); err != nil {
		t.Fatal(err)
	}
	for _, s := range payload.Steps {
		if s.Status == "blocked" && len(s.BlockerIDs) == 0 {
			t.Errorf("blocked step %q names no blockers", s.ID)
		}
		if len(s.BlockerIDs) == 0 {
			continue
		}
		args, _ := json.Marshal(map[string]string{"project_id": "1", "step_id": s.ID})
		stepOut, err := handleGetStep(st, context.Background(), args)
		if err != nil {
			t.Fatalf("get_step(%s) error = %v", s.ID, err)
		}
		for _, bid := range s.BlockerIDs {
			if !strings.Contains(stepOut, `"`+bid+`"`) {
				t.Errorf("blocker %q from list_steps not found in get_step(%s)", bid, s.ID)
			}
		}
	}
}
