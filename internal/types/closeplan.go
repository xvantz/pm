package types

// ClosePlanStep is one step that a bulk close would move to done.
type ClosePlanStep struct {
	ID     string     `json:"id" yaml:"id"`
	Title  string     `json:"title" yaml:"title"`
	Status StepStatus `json:"status" yaml:"status"`
}

// ClosePlanBlocker is an unresolved blocker on a step the close would complete.
//
// Closing does NOT resolve blockers: CloseProject moves the step to done and
// leaves the blocker records as they were. A blocker sitting on a completed
// step is dead weight the project can still be reopened from, so a human
// confirming the close is told plainly that the blockers stay on the record.
type ClosePlanBlocker struct {
	ID       string `json:"id" yaml:"id"`
	Title    string `json:"title" yaml:"title"`
	Reason   string `json:"reason,omitempty" yaml:"reason,omitempty"`
	StepID   string `json:"step_id" yaml:"step_id"`
	StepName string `json:"step_title" yaml:"step_title"`
}

// ClosePlan is a read-only preview of what CloseProject would do.
//
// It exists because bulk close is the one operation that can mark unfinished
// work finished: it bypasses the step lifecycle on purpose. A caller can show
// this to a human before performing the irreversible half.
type ClosePlan struct {
	ProjectID    string             `json:"project_id" yaml:"project_id"`
	ProjectTitle string             `json:"project_title" yaml:"project_title"`
	Steps        []ClosePlanStep    `json:"steps" yaml:"steps"`
	Blockers     []ClosePlanBlocker `json:"blockers,omitempty" yaml:"blockers,omitempty"`
}

// StepsToClose is the number of steps this plan would move to done.
func (p ClosePlan) StepsToClose() int { return len(p.Steps) }

// BlockersToResolve is the number of blockers this plan would complete.
func (p ClosePlan) BlockersToResolve() int { return len(p.Blockers) }

// Empty reports that there is nothing left to close: every step is already
// done. Not an error — it just means the close is a formality.
func (p ClosePlan) Empty() bool { return len(p.Steps) == 0 }
