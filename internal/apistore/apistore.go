// Package apistore implements store.Store over HTTP against `pm serve`.
//
// It lets CLI and MCP run with zero handler changes: set PM_API to the
// daemon address (and PM_TOKEN) and every operation goes through the
// single-writer daemon instead of YAML files.
//
// Semantics notes:
//   - NextNumber is advisory (max+1 from the list); the daemon assigns the
//     real number server-side under its mutex. Concurrent creates may print
//     a stale number in confirmation texts but never duplicate one.
//   - AdvanceNextNumber is a no-op: the daemon advances its counter on create.
//   - SaveStep/SaveBlocker translate entity deltas into lifecycle endpoints
//     (start/review/done/resolve) so server-side validation always applies.
//     A blocked→todo write after resolve is a no-op: the daemon already
//     unblocked the step when resolving.
package apistore

import (
	"fmt"

	"github.com/xvantz/pm/internal/client"
	"github.com/xvantz/pm/internal/types"
)

// Store is a store.Store speaking to `pm serve`.
type Store struct {
	c *client.Client
}

// New returns a Store for the given client.
func New(c *client.Client) *Store { return &Store{c: c} }

// NewFromEnv returns a Store when PM_API is set, or (nil, false) otherwise.
// PM_TOKEN carries the Bearer token (may be empty for a tokenless daemon,
// which rejects everything except /healthz).
func NewFromEnv() (*Store, bool) {
	return NewFromEnvVars(envAPI(), envToken())
}

func (s *Store) ListProjects() ([]types.Project, error) {
	return s.c.ListProjects()
}

func (s *Store) GetProject(id string) (*types.ProjectData, error) {
	return s.c.GetProject(id)
}

func (s *Store) ResolveProject(ref string) (*types.ProjectData, error) {
	return s.c.GetProject(ref)
}

func (s *Store) NextNumber() (int, error) {
	projects, err := s.c.ListProjects()
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

func (s *Store) AdvanceNextNumber() error { return nil }

func (s *Store) GetSteps(projectID string) ([]types.Step, error) {
	return s.c.ListSteps(projectID)
}

func (s *Store) GetBlockers(projectID string) ([]types.Blocker, error) {
	return s.c.ListBlockers(projectID)
}

func (s *Store) GetDecisions(projectID string) ([]types.Decision, error) {
	return s.c.ListDecisions(projectID)
}

func (s *Store) SaveProject(p types.Project) error {
	// New vs existing by lookup: callers (CLI) pre-assign Number before save,
	// so Number==0 is not a reliable new-project signal.
	if _, err := s.c.GetProject(p.ID); err != nil {
		_, cerr := s.c.CreateProject(p.Title, p.Goal, p.Tags, p.ID)
		return cerr
	}
	_, err := s.c.PatchProject(p.ID, &p.Goal, &p.Status, p.Tags)
	return err
}

func (s *Store) SaveStep(st types.Step) error {
	steps, err := s.c.ListSteps(st.ProjectID)
	if err != nil {
		return err
	}
	var cur *types.Step
	for i := range steps {
		if steps[i].ID == st.ID {
			cur = &steps[i]
			break
		}
	}
	if cur == nil {
		_, err := s.c.AddStep(st.ProjectID, st.Title)
		return err
	}
	if cur.Status == st.Status {
		return nil
	}
	action, err := stepAction(cur.Status, st.Status)
	if err != nil {
		return err
	}
	if action == "" {
		return nil // server already applied it (e.g. unblock on resolve)
	}
	_, err = s.c.StepAction(st.ProjectID, st.ID, action)
	return err
}

// stepAction maps a status delta to a lifecycle endpoint.
func stepAction(from, to types.StepStatus) (string, error) {
	switch {
	case to == types.StepInProgress && from == types.StepTodo:
		return "start", nil
	case to == types.StepReview && (from == types.StepTodo || from == types.StepInProgress):
		return "review", nil
	case to == types.StepDone && from == types.StepReview:
		return "done", nil
	case to == types.StepTodo && from == types.StepBlocked:
		return "", nil
	default:
		return "", fmt.Errorf("step is %s, cannot move to %s", from, to)
	}
}

func (s *Store) SaveBlocker(b types.Blocker) error {
	steps, err := s.c.ListSteps(b.ProjectID)
	if err != nil {
		return err
	}
	for _, st := range steps {
		if st.ID != b.StepID {
			continue
		}
		for _, existing := range st.Blockers {
			if existing.ID == b.ID {
				if b.Status == types.BlockerResolved && existing.Status != types.BlockerResolved {
					_, err := s.c.ResolveBlocker(b.ProjectID, b.StepID, b.ID)
					return err
				}
				return nil
			}
		}
		_, err := s.c.AddBlocker(b.ProjectID, b.StepID, b.Title, b.Reason)
		return err
	}
	return fmt.Errorf("step %q not found", b.StepID)
}

func (s *Store) SaveDecision(d types.Decision) error {
	decisions, err := s.c.ListDecisions(d.ProjectID)
	if err != nil {
		return err
	}
	for _, existing := range decisions {
		if existing.ID == d.ID {
			return nil // decisions are immutable
		}
	}
	_, err = s.c.AddDecision(d.ProjectID, d.Title, d.Reason)
	return err
}

func (s *Store) DeleteProject(id string) error {
	_, err := s.c.DeleteProject(id)
	return err
}

// CloseProject forwards the consent flag to the daemon unchanged. The gate
// itself lives in the store and the daemon, so an adapter cannot skip it.
//
// The HTTP call answers with either a plan or a closed project depending on
// confirm; here both collapse into (plan, error) because that is what the
// store contract promises. The caller learns the outcome from the store's
// state, not from this return value.
func (s *Store) CloseProject(ref, reason string, confirm bool) (*types.ClosePlan, error) {
	_, plan, err := s.c.CloseProject(ref, reason, confirm)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// ClosePlan previews a close over the daemon without touching anything.
func (s *Store) ClosePlan(ref string) (*types.ClosePlan, error) {
	return s.c.ClosePlan(ref)
}

func (s *Store) TrashList() ([]string, error) {
	return s.c.TrashList()
}

func (s *Store) TrashRestore(trashName string) error {
	return s.c.TrashRestore(trashName)
}

func (s *Store) TrashClean() error {
	return s.c.TrashClean()
}

func (s *Store) DeleteStep(projectID, stepID string) error {
	return s.c.DeleteStep(projectID, stepID)
}

func (s *Store) DeleteBlocker(projectID, stepID, blockerID string) error {
	return s.c.DeleteBlocker(projectID, stepID, blockerID)
}

func (s *Store) DeleteDecision(projectID, decisionID string) error {
	return s.c.DeleteDecision(projectID, decisionID)
}
