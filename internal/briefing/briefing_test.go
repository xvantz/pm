package briefing

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xvantz/pm/internal/store"
)

func TestGenerate_Basic(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-14"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if b.Date != "2026-06-14" {
		t.Errorf("Date = %q, want %q", b.Date, "2026-06-14")
	}

	if b.Summary.TotalProjects == 0 {
		t.Error("Summary.TotalProjects = 0, expected > 0")
	}

	if len(b.Sections) == 0 {
		t.Error("Sections empty, expected at least one section")
	}

	if len(b.Recommendations) == 0 {
		t.Error("Recommendations empty, expected at least one recommendation")
	}
}

func TestGenerate_SummaryCounts(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-14"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if b.Summary.ActiveProjects < 2 {
		t.Errorf("ActiveProjects = %d, want >= 2", b.Summary.ActiveProjects)
	}

	if b.Summary.IdeaProjects < 1 {
		t.Errorf("IdeaProjects = %d, want >= 1", b.Summary.IdeaProjects)
	}
}

func TestGenerate_MarkdownOutput(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-14"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	md := b.FormatMarkdown()

	checks := []string{
		"PM Briefing",
		"Активных проектов",
		"Рекомендации на сегодня",
		"Сгенерировано",
	}

	for _, c := range checks {
		if !strings.Contains(md, c) {
			t.Errorf("Markdown missing expected text: %q", c)
		}
	}
}

func TestGenerate_JSONRoundtrip(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-14"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Briefing
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Date != "2026-06-14" {
		t.Errorf("JSON roundtrip Date = %q, want %q", decoded.Date, "2026-06-14")
	}
}

func TestGenerate_BlockedSection(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-14"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	foundBlocked := false
	for _, sec := range b.Sections {
		if sec.Type == "blocked" {
			foundBlocked = true
			break
		}
	}

	if !foundBlocked {
		t.Error("Blocked section not found, but mock has blocked projects")
	}
}

func TestGenerate_SpecificDate(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-13"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if b.Date != "2026-06-13" {
		t.Errorf("Date = %q, want %q", b.Date, "2026-06-13")
	}
}

func TestGenerate_TodayFallback(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	expected := time.Now().UTC().Format("2006-01-02")
	if b.Date != expected {
		t.Errorf("Date = %q, want today %q", b.Date, expected)
	}
}

func TestGenerate_StepProgress(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-14"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// PM project: #2, 7 steps, should have some done
	found := false
	for _, sec := range b.Sections {
		for _, p := range sec.Projects {
			if p.ID == "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b" {
				found = true
				if p.StepsTotal != 7 {
					t.Errorf("PM StepsTotal = %d, want 7", p.StepsTotal)
				}
				if p.StepsDone < 1 {
					t.Errorf("PM StepsDone = %d, want >= 1", p.StepsDone)
				}
			}
		}
	}
	if !found {
		t.Error("PM project not found in any section")
	}
}

func TestGenerate_EmptyStore(t *testing.T) {
	st := store.NewMockStore()
	raw := st.GetRaw()
	for id := range raw {
		delete(raw, id)
	}

	cfg := Config{Store: st}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if b.Summary.TotalProjects != 0 {
		t.Errorf("TotalProjects = %d, want 0", b.Summary.TotalProjects)
	}
}

func TestGenerate_RecommendationsPriority(t *testing.T) {
	st := store.NewMockStore()
	cfg := Config{Store: st, Date: "2026-06-14"}

	b, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	for i := 1; i < len(b.Recommendations); i++ {
		if b.Recommendations[i].Priority <= b.Recommendations[i-1].Priority {
			t.Errorf("Recommendation %d has priority %d <= previous %d",
				i, b.Recommendations[i].Priority, b.Recommendations[i-1].Priority)
		}
	}
}

func TestGenerate_WithNumericReference(t *testing.T) {
	st := store.NewMockStore()
	// Resolve by number should work
	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("ResolveProject(1) error = %v", err)
	}
	if pd == nil {
		t.Fatal("ResolveProject(1) returned nil")
	}
	if pd.Project.Title != "AdGuard Home" {
		t.Errorf("Project #1 title = %q, want %q", pd.Project.Title, "AdGuard Home")
	}
}

func TestGenerate_ProjectsHaveNumbers(t *testing.T) {
	st := store.NewMockStore()
	projects, err := st.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	for _, p := range projects {
		if p.Number == 0 {
			t.Errorf("Project %q has Number = 0", p.ID)
		}
	}
}
