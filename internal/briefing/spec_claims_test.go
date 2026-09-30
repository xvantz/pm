package briefing

import (
	"testing"
	"time"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// The spec claims these behaviours; this file proves the ones nothing else
// covers, so the spec cannot drift away from the code unnoticed.

// steps_today and projects_moved are distinct: several steps in one project
// move that project once.
func TestSpec_ManyStepsMoveOneProjectOnce(t *testing.T) {
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if b.Summary.StepsToday < 2 {
		t.Fatalf("need at least two steps today to test the distinction, got %d", b.Summary.StepsToday)
	}
	if b.Summary.ProjectsMoved != 1 {
		t.Errorf("ProjectsMoved = %d, want 1 for a project with %d steps done today",
			b.Summary.ProjectsMoved, b.Summary.StepsToday)
	}
	if b.Summary.StepsToday <= b.Summary.ProjectsMoved {
		t.Errorf("steps (%d) and projects (%d) must be counted separately",
			b.Summary.StepsToday, b.Summary.ProjectsMoved)
	}
}

// A blocked project appears in blocked and NOT in active.
func TestSpec_BlockedProjectIsNotAlsoActive(t *testing.T) {
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	active := map[string]bool{}
	for _, p := range b.Sections {
		if p.Type != "active" {
			continue
		}
		for _, ps := range p.Projects {
			active[ps.ID] = true
		}
	}
	for _, sec := range b.Sections {
		if sec.Type != "blocked" {
			continue
		}
		for _, ps := range sec.Projects {
			if active[ps.ID] {
				t.Errorf("project %q (%s) is in both active and blocked", ps.ID, ps.Title)
			}
		}
	}
}

// Sections appear in the documented order.
func TestSpec_SectionOrderIsStable(t *testing.T) {
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	want := []string{"active", "blocked", "idea", "completed"}
	got := make([]string, 0, len(b.Sections))
	for _, s := range b.Sections {
		got = append(got, s.Type)
		if len(s.Projects) == 0 {
			t.Errorf("empty section %q is emitted; it should be omitted", s.Type)
		}
	}
	// got must be want filtered to the types actually present.
	var expect []string
	present := map[string]bool{}
	for _, s := range b.Sections {
		present[s.Type] = true
	}
	for _, w := range want {
		if present[w] {
			expect = append(expect, w)
		}
	}
	if len(got) != len(expect) {
		t.Fatalf("sections = %v, want %v (order and membership)", got, expect)
	}
	for i := range expect {
		if got[i] != expect[i] {
			t.Errorf("section %d = %q, want %q (order)", i, got[i], expect[i])
		}
	}
}

// A fresh blocker must not show up as long-lived.
func TestSpec_FreshBlockerIsNotLongLived(t *testing.T) {
	st := store.NewMockStore()
	// The seeded blocker was created 2026-06-10; a briefing for 2026-06-14 is
	// four days later, under the seven-day threshold.
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	for _, bl := range b.Summary.LongLivedBlockers {
		if bl.DaysAlive <= 7 {
			t.Errorf("blocker %q listed as long-lived at %d days", bl.BlockerTitle, bl.DaysAlive)
		}
	}
}

// Priorities are sequential and dense.
func TestSpec_RecommendationPrioritiesAreDense(t *testing.T) {
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	for i, r := range b.Recommendations {
		if r.Priority != i+1 {
			t.Errorf("recommendation %d has priority %d, want %d (dense, starting at 1)",
				i, r.Priority, i+1)
		}
	}
}

// The single-project filter narrows every count.
func TestSpec_SingleProjectFilterNarrowsEverything(t *testing.T) {
	st := store.NewMockStore()
	full, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("full Generate error = %v", err)
	}
	if full.Summary.TotalProjects < 2 {
		t.Skip("need more than one seeded project to test the filter")
	}
	one, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC, FilterProject: "1"})
	if err != nil {
		t.Fatalf("filtered Generate error = %v", err)
	}
	if one.Summary.TotalProjects != 1 {
		t.Errorf("filtered TotalProjects = %d, want 1", one.Summary.TotalProjects)
	}
	total := 0
	for _, s := range one.Sections {
		for _, p := range s.Projects {
			total++
			if p.ID != one.Sections[0].Projects[0].ID {
				t.Errorf("filter leaked project %q", p.Title)
			}
		}
	}
	if total != 1 {
		t.Errorf("filtered briefing shows %d projects across sections, want 1", total)
	}
}

// An unparsable requested day falls back to today instead of failing.
func TestSpec_UnparsableDayFallsBack(t *testing.T) {
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "not-a-date", Loc: time.UTC})
	if err != nil {
		t.Fatalf("an unparsable day must not fail the briefing, got: %v", err)
	}
	want := time.Now().UTC().Format("2006-01-02")
	if b.Date != want {
		t.Errorf("Date = %q, want the current day %q", b.Date, want)
	}
}

// Paused projects are counted but appear in no section.
func TestSpec_PausedIsCountedNotShown(t *testing.T) {
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if b.Summary.PausedProjects == 0 {
		t.Skip("no paused project in the seed")
	}
	for _, sec := range b.Sections {
		if sec.Type == "paused" {
			t.Error("a paused section exists; paused projects are counted, not shown")
		}
	}
}

var _ = types.StepDone // keep the types import honest for readers of this file
