package domain

import (
	"testing"

	"github.com/xvantz/pm/internal/types"
)

func TestValidateStepStart(t *testing.T) {
	// Todo → OK
	err := ValidateStepStart(types.Step{ID: "s1", Status: types.StepTodo})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	// InProgress → error
	err = ValidateStepStart(types.Step{ID: "s1", Status: types.StepInProgress})
	if err == nil {
		t.Error("expected error for in_progress step")
	}
	// Done → error
	err = ValidateStepStart(types.Step{ID: "s1", Status: types.StepDone})
	if err == nil {
		t.Error("expected error for done step")
	}
}

func TestValidateStepReview(t *testing.T) {
	// Todo → OK
	err := ValidateStepReview(types.Step{ID: "s1", Status: types.StepTodo})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	// InProgress → OK
	err = ValidateStepReview(types.Step{ID: "s1", Status: types.StepInProgress})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	// Done → error
	err = ValidateStepReview(types.Step{ID: "s1", Status: types.StepDone})
	if err == nil {
		t.Error("expected error for done step")
	}
}

func TestValidateStepDone(t *testing.T) {
	// Review → OK
	err := ValidateStepDone(types.Step{ID: "s1", Status: types.StepReview})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	// Todo → error
	err = ValidateStepDone(types.Step{ID: "s1", Status: types.StepTodo})
	if err == nil {
		t.Error("expected error for todo step")
	}
	// Review with unresolved blocker → error
	err = ValidateStepDone(types.Step{
		ID: "s1", Status: types.StepReview,
		Blockers: []types.Blocker{{Status: types.BlockerActive}},
	})
	if err == nil {
		t.Error("expected error for step with unresolved blockers")
	}
	// Review with only resolved blockers → OK
	err = ValidateStepDone(types.Step{
		ID: "s1", Status: types.StepReview,
		Blockers: []types.Blocker{{Status: types.BlockerResolved}},
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestHasUnresolvedBlockers(t *testing.T) {
	if HasUnresolvedBlockers(nil) {
		t.Error("nil blockers should not be unresolved")
	}
	if HasUnresolvedBlockers([]types.Blocker{}) {
		t.Error("empty blockers should not be unresolved")
	}
	if HasUnresolvedBlockers([]types.Blocker{{Status: types.BlockerResolved}}) {
		t.Error("resolved blockers should not be unresolved")
	}
	if !HasUnresolvedBlockers([]types.Blocker{{Status: types.BlockerActive}}) {
		t.Error("active blockers should be unresolved")
	}
	if !HasUnresolvedBlockers([]types.Blocker{{Status: types.BlockerWaiting}}) {
		t.Error("waiting blockers should be unresolved")
	}
}

func TestStepStatusChange(t *testing.T) {
	now := "2025-01-01"
	s := &types.Step{ID: "s1", Status: types.StepTodo, UpdatedAt: "2024-01-01"}
	StepStatusChange(s, types.StepInProgress, now)
	if s.Status != types.StepInProgress {
		t.Errorf("expected in_progress, got %s", s.Status)
	}
	if s.UpdatedAt != now {
		t.Errorf("expected %s, got %s", now, s.UpdatedAt)
	}
}
