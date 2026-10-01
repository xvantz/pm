package types

// DoctorProjectLine is one project's row in an integrity report: enough to
// show the human what was checked, not enough to act on.
type DoctorProjectLine struct {
	Number    int    `json:"number" yaml:"number"`
	Title     string `json:"title" yaml:"title"`
	ID        string `json:"id" yaml:"id"`
	Steps     int    `json:"steps" yaml:"steps"`
	Blockers  int    `json:"blockers" yaml:"blockers"`
	Decisions int    `json:"decisions" yaml:"decisions"`
}

// DoctorReport is the result of an integrity walk over one store.
//
// It is produced by the single writer (the daemon owns the files), so every
// caller observes the same state. Counts cover entities; Issues names every
// concrete problem found (read errors, parse errors, id mismatches).
// LegacyTimestamps is a progress counter, not a repair list: date-only stamps
// are rewritten in canonical form on the next write of their record.
type DoctorReport struct {
	Root             string              `json:"root" yaml:"root"`
	Projects         []DoctorProjectLine `json:"projects" yaml:"projects"`
	TotalSteps       int                 `json:"total_steps" yaml:"total_steps"`
	TotalBlockers    int                 `json:"total_blockers" yaml:"total_blockers"`
	TotalDecisions   int                 `json:"total_decisions" yaml:"total_decisions"`
	Orphans          []string            `json:"orphans,omitempty" yaml:"orphans,omitempty"`
	Issues           []string            `json:"issues,omitempty" yaml:"issues,omitempty"`
	LegacyTimestamps int                 `json:"legacy_timestamps" yaml:"legacy_timestamps"`
	BrokenTimestamps int                 `json:"broken_timestamps" yaml:"broken_timestamps"`
}

// HasIssues reports whether the walk found anything that needs a human.
func (r DoctorReport) HasIssues() bool {
	return len(r.Issues) > 0 || len(r.Orphans) > 0
}
