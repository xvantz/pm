package types

import (
	"testing"
)

func TestProjectDefaults(t *testing.T) {
	p := Project{
		ID: "test", Title: "Test",
		Status: StatusActive,
	}

	if p.ID != "test" {
		t.Errorf("ID = %q, want %q", p.ID, "test")
	}

	if p.Status != StatusActive {
		t.Errorf("Status = %q, want %q", p.Status, StatusActive)
	}

	if p.CreatedAt != "" {
		t.Errorf("CreatedAt should be empty, got %q", p.CreatedAt)
	}
}

func TestStepStatuses(t *testing.T) {
	tests := []struct {
		status StepStatus
		want   string
	}{
		{StepTodo, "todo"},
		{StepInProgress, "in_progress"},
		{StepReview, "review"},
		{StepDone, "done"},
		{StepBlocked, "blocked"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("StepStatus(%s) = %q, want %q", tt.want, string(tt.status), tt.want)
		}
	}
}

func TestProjectStatuses(t *testing.T) {
	tests := []struct {
		status ProjectStatus
		want   string
	}{
		{StatusActive, "active"},
		{StatusCompleted, "completed"},
		{StatusPaused, "paused"},
		{StatusIdea, "idea"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("ProjectStatus(%s) = %q, want %q", tt.want, string(tt.status), tt.want)
		}
	}
}

func TestBlockerStatuses(t *testing.T) {
	tests := []struct {
		status BlockerStatus
		want   string
	}{
		{BlockerWaiting, "waiting"},
		{BlockerActive, "active"},
		{BlockerResolved, "resolved"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("BlockerStatus(%s) = %q, want %q", tt.want, string(tt.status), tt.want)
		}
	}
}

func TestStepArtifacts(t *testing.T) {
	s := Step{
		ID: "setup-caddy", Title: "Setup Caddy",
		Status: StepDone, ProjectID: "agh",
		Artifacts: []string{
			"github.com/user/repo/pull/15",
			"docs/architecture.md",
		},
	}

	if len(s.Artifacts) != 2 {
		t.Errorf("Artifacts length = %d, want 2", len(s.Artifacts))
	}

	if s.Artifacts[0] != "github.com/user/repo/pull/15" {
		t.Errorf("Artifact[0] = %q, want %q", s.Artifacts[0], "github.com/user/repo/pull/15")
	}
}

func TestProjectTags(t *testing.T) {
	p := Project{
		ID: "agh", Title: "AdGuard Home",
		Status: StatusActive,
		Tags:   []string{"infrastructure", "homelab", "networking"},
	}

	if len(p.Tags) != 3 {
		t.Errorf("Tags length = %d, want 3", len(p.Tags))
	}

	if p.Tags[0] != "infrastructure" {
		t.Errorf("Tag[0] = %q, want %q", p.Tags[0], "infrastructure")
	}
}

func TestProjectDataComposition(t *testing.T) {
	pd := ProjectData{
		Project: Project{
			ID: "test", Title: "Test Project",
			Status: StatusActive,
		},
		Steps: []Step{
			{ID: "step1", Title: "Step 1", Status: StepDone, ProjectID: "test"},
			{ID: "step2", Title: "Step 2", Status: StepBlocked, ProjectID: "test",
				Blockers: []Blocker{
					{ID: "block1", Title: "Blocker 1", Status: BlockerActive, ProjectID: "test", StepID: "step2"},
				},
			},
		},
		Decisions: []Decision{
			{ID: "dec1", Title: "Decision 1", Date: "2026-06-14", ProjectID: "test"},
		},
	}

	if len(pd.Steps) != 2 {
		t.Errorf("Steps count = %d, want 2", len(pd.Steps))
	}

	// Blockers are inside steps now
	totalBlockers := 0
	for _, s := range pd.Steps {
		totalBlockers += len(s.Blockers)
	}
	if totalBlockers != 1 {
		t.Errorf("Total blockers = %d, want 1", totalBlockers)
	}

	if len(pd.Decisions) != 1 {
		t.Errorf("Decisions count = %d, want 1", len(pd.Decisions))
	}

	if pd.Project.ID != "test" {
		t.Errorf("Project ID = %q, want %q", pd.Project.ID, "test")
	}
}
