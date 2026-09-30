package briefing

import (
	"testing"
	"time"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// Both surfaces must agree on "today". They can only agree if the zone comes
// from one place: the daemon process. This test pins that contract, because a
// silent divergence here means the CLI and an agent reading the same store get
// different numbers for the same question.
func TestGenerate_SurfacesAgreeOnTheSameZone(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	st := store.NewMockStore()
	day := "2026-06-14"

	// Simulate the daemon reading in its own zone.
	daemonSide, err := Generate(Config{Store: st, Date: day, Loc: moscow})
	if err != nil {
		t.Fatalf("daemon side: %v", err)
	}
	// And a caller that passes no Loc — the default every real surface uses.
	defaultSide, err := Generate(Config{Store: st, Date: day})
	if err != nil {
		t.Fatalf("default side: %v", err)
	}

	if daemonSide.Summary.StepsToday != defaultSide.Summary.StepsToday {
		t.Errorf("StepsToday differs: explicit zone %d, default %d",
			daemonSide.Summary.StepsToday, defaultSide.Summary.StepsToday)
	}
	if daemonSide.Summary.ProjectsMoved != defaultSide.Summary.ProjectsMoved {
		t.Errorf("ProjectsMoved differs: %d vs %d",
			daemonSide.Summary.ProjectsMoved, defaultSide.Summary.ProjectsMoved)
	}
	if daemonSide.Date != defaultSide.Date {
		t.Errorf("Date differs: %q vs %q", daemonSide.Date, defaultSide.Date)
	}
}

// Config.Loc must default to the process zone, so the daemon's TZ is the only
// input. Guarded explicitly because the whole zone decision rests on it.
func TestConfig_LocDefaultsToProcessZone(t *testing.T) {
	var c Config
	if c.loc() != time.Local {
		t.Errorf("Config.loc() = %v, want time.Local", c.loc())
	}
	fixed := time.FixedZone("MSK", 3*60*60)
	c.Loc = fixed
	if c.loc() != fixed {
		t.Errorf("Config.loc() = %v, want the explicit zone", c.loc())
	}
}

func TestGenerate_BothZonesCountTheirOwnDay(t *testing.T) {
	// A step closed at 23:30 MSK on the 14th is 20:30 UTC on the 14th, but at
	// 03:00 MSK on the 15th it is 00:00 UTC on the 15th. Whichever zone the
	// daemon runs in must be the zone the count follows.
	steps := []types.Step{{
		ID: "late", Status: types.StepDone,
		UpdatedAt: mustTS("2026-09-15T00:30:00+03:00"), // 2026-09-14T21:30Z
	}}
	if got := countStepsOnDay(steps, "2026-09-15", types.StepDone, time.UTC, &Briefing{}); got != 0 {
		t.Errorf("UTC: count = %d, want 0 (it is the 14th in UTC)", got)
	}
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	if got := countStepsOnDay(steps, "2026-09-15", types.StepDone, moscow, &Briefing{}); got != 1 {
		t.Errorf("MSK: count = %d, want 1 (it is the 15th locally)", got)
	}
}
