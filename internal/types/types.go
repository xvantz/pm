package types

import "time"

type ProjectStatus string

const (
	StatusActive    ProjectStatus = "active"
	StatusCompleted ProjectStatus = "completed"
	StatusPaused    ProjectStatus = "paused"
	StatusIdea      ProjectStatus = "idea"
)

type StepStatus string

const (
	StepTodo       StepStatus = "todo"
	StepInProgress StepStatus = "in_progress"
	StepReview     StepStatus = "review"
	StepDone       StepStatus = "done"
	StepBlocked    StepStatus = "blocked"
)

type BlockerStatus string

const (
	BlockerWaiting  BlockerStatus = "waiting"
	BlockerActive   BlockerStatus = "active"
	BlockerResolved BlockerStatus = "resolved"
)

type Project struct {
	ID          string        `yaml:"id" json:"id"`
	Number      int           `yaml:"number" json:"number"`
	Title       string        `yaml:"title" json:"title"`
	Goal        string        `yaml:"goal,omitempty" json:"goal,omitempty"`
	Status      ProjectStatus `yaml:"status" json:"status"`
	Tags        []string      `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt   string        `yaml:"created_at" json:"created_at"`
	UpdatedAt   string        `yaml:"updated_at" json:"updated_at"`
	CompletedAt string        `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
}

type Step struct {
	ID        string     `yaml:"id" json:"id"`
	Title     string     `yaml:"title" json:"title"`
	Status    StepStatus `yaml:"status" json:"status"`
	ProjectID string     `yaml:"project_id" json:"project_id"`
	Blockers  []Blocker  `yaml:"blockers,omitempty" json:"blockers,omitempty"`
	Artifacts []string   `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Deps      []string   `yaml:"deps,omitempty" json:"deps,omitempty"`
	CreatedAt string     `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt string     `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

type Blocker struct {
	ID        string         `yaml:"id" json:"id"`
	Title     string         `yaml:"title" json:"title"`
	Reason    string         `yaml:"reason,omitempty" json:"reason,omitempty"`
	Status    BlockerStatus  `yaml:"status" json:"status"`
	ProjectID string         `yaml:"project_id" json:"project_id"`
	StepID    string         `yaml:"step_id" json:"step_id"`
	CreatedAt string         `yaml:"created_at,omitempty" json:"created_at,omitempty"`
}

type Decision struct {
	ID        string `yaml:"id" json:"id"`
	Title     string `yaml:"title" json:"title"`
	Reason    string `yaml:"reason,omitempty" json:"reason,omitempty"`
	Date      string `yaml:"date" json:"date"`
	ProjectID string `yaml:"project_id" json:"project_id"`
}

type ProjectData struct {
	Project   Project    `json:"project"`
	Steps     []Step     `json:"steps"`
	Decisions []Decision `json:"decisions"`
}

func NowISO() string {
	return time.Now().UTC().Format("2006-01-02")
}
