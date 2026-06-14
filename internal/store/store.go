package store

import "github.com/xvantz/pm/internal/types"

type Store interface {
	// ListProjects returns all projects.
	ListProjects() ([]types.Project, error)
	// GetProject returns a single project by internal UUID.
	GetProject(id string) (*types.ProjectData, error)
	// ResolveProject finds a project by display number (as string) or internal UUID.
	ResolveProject(ref string) (*types.ProjectData, error)
	// NextNumber returns the next sequential project number.
	NextNumber() (int, error)
	// GetSteps returns all steps for a project.
	GetSteps(projectID string) ([]types.Step, error)
	// GetBlockers returns all blockers for a project.
	GetBlockers(projectID string) ([]types.Blocker, error)
	// GetDecisions returns all decisions for a project.
	GetDecisions(projectID string) ([]types.Decision, error)
	// SaveProject creates or updates a project.
	SaveProject(p types.Project) error
	// SaveStep creates or updates a step.
	SaveStep(s types.Step) error
	// SaveBlocker creates or updates a blocker.
	SaveBlocker(b types.Blocker) error
	// SaveDecision creates or updates a decision.
	SaveDecision(d types.Decision) error
	// DeleteProject removes a project and all its data.
	DeleteProject(id string) error
	// DeleteStep removes a step and its blockers.
	DeleteStep(projectID, stepID string) error
	// DeleteBlocker removes a blocker from a step.
	DeleteBlocker(projectID, stepID, blockerID string) error
	// DeleteDecision removes a decision.
	DeleteDecision(projectID, decisionID string) error
}
