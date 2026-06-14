package domain

import (
	"fmt"

	"github.com/xvantz/pm/internal/types"
)

// ValidateStepStart checks if a step can be started (moved from todo to in_progress).
func ValidateStepStart(s types.Step) error {
	if s.Status != types.StepTodo {
		return fmt.Errorf("step %q is %s, can only start from todo", s.ID, s.Status)
	}
	return nil
}

// ValidateStepReview checks if a step can be sent to review.
func ValidateStepReview(s types.Step) error {
	if s.Status != types.StepTodo && s.Status != types.StepInProgress {
		return fmt.Errorf("step %q is %s, can only review todo or in_progress steps", s.ID, s.Status)
	}
	return nil
}

// ValidateStepDone checks if a step can be marked done.
func ValidateStepDone(s types.Step) error {
	if s.Status != types.StepReview {
		return fmt.Errorf("step %q is %s, must be in review before done", s.ID, s.Status)
	}
	// Check for unresolved blockers
	for _, b := range s.Blockers {
		if b.Status == types.BlockerActive || b.Status == types.BlockerWaiting {
			return fmt.Errorf("step %q has unresolved blockers — resolve them first", s.ID)
		}
	}
	return nil
}

// HasUnresolvedBlockers returns true if any blocker in the list is active or waiting.
func HasUnresolvedBlockers(blockers []types.Blocker) bool {
	for _, b := range blockers {
		if b.Status == types.BlockerActive || b.Status == types.BlockerWaiting {
			return true
		}
	}
	return false
}

// StepStatusChange applies a new status to a step after validation, and updates UpdatedAt.
func StepStatusChange(s *types.Step, newStatus types.StepStatus, now string) {
	s.Status = newStatus
	s.UpdatedAt = now
}
