package briefing

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// mustTS builds a typed timestamp for a fixture.
func mustTS(s string) types.Timestamp {
	ts, err := types.ParseTimestamp(s)
	if err != nil {
		panic(err)
	}
	return ts
}

// mustBroken builds a known-unreadable timestamp, for the warning path.
func mustBroken(s string) types.Timestamp { return types.NewInvalidTimestamp(s) }

func TestGenerate_CountsIntradaySteps(t *testing.T) {
	// The regression this change exists for: the old code compared
	// `s.UpdatedAt == "2026-06-14"`, which matches nothing once timestamps
	// are RFC3339. Two steps finished on the same day used to report zero,
	// with no error anywhere. The mock carries two intraday stamps precisely
	// so this cannot regress silently again.
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if b.Summary.StepsToday != 2 {
		t.Errorf("StepsToday = %d, want 2 (two done steps on 2026-06-14)", b.Summary.StepsToday)
	}
	if b.Summary.StepsThisWeek != 2 {
		t.Errorf("StepsThisWeek = %d, want 2", b.Summary.StepsThisWeek)
	}
	if b.Summary.ProjectsMoved != 1 {
		t.Errorf("ProjectsMoved = %d, want 1", b.Summary.ProjectsMoved)
	}
}

func TestGenerate_LegacyDateStepsStillCount(t *testing.T) {
	// A project still carrying the legacy date form must keep contributing:
	// read compatibility is the whole reason the migration can be manual.
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-12", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	// Navidrome was created 2026-06-12 with no done steps, so this is a
	// sanity check that the legacy path is parsed, not that a number is big.
	for _, w := range b.DataWarnings {
		if strings.Contains(w, "unparseable") {
			t.Errorf("legacy date reported unparseable: %q", w)
		}
	}
}

func TestGenerate_CalendarDayFollowsReaderZone(t *testing.T) {
	// 01:00 MSK is 22:00 UTC on the PREVIOUS day. Bucketing in UTC files it
	// under yesterday; the daemon's zone is the one that counts.
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}

	steps := []types.Step{{
		ID: "night-shift", Title: "Закрыт в час ночи", Status: types.StepDone,
		UpdatedAt: mustTS("2026-09-29T01:00:00+03:00"), // = 2026-09-28T22:00:00Z
	}}
	loc := mustLoc(t, moscow)

	gotLocal := countStepsOnDay(steps, "2026-09-29", types.StepDone, loc, &Briefing{})
	if gotLocal != 1 {
		t.Errorf("local zone: count = %d, want 1 (the user's today, not UTC's yesterday)", gotLocal)
	}
	gotUTC := countStepsOnDay(steps, "2026-09-28", types.StepDone, time.UTC, &Briefing{})
	if gotUTC != 1 {
		t.Errorf("UTC zone: count = %d, want 1 for the previous UTC day", gotUTC)
	}
	// And the key point: in the local zone it must NOT appear on the 28th.
	if cross := countStepsOnDay(steps, "2026-09-28", types.StepDone, loc, &Briefing{}); cross != 0 {
		t.Errorf("local zone counted it on 2026-09-28 (%d), want 0", cross)
	}
}

func mustLoc(t *testing.T, l *time.Location) *time.Location {
	t.Helper()
	return l
}

func TestGenerate_UnparseableTimestampWarnsInsteadOfLying(t *testing.T) {
	// The loud-failure contract: a broken stamp must not silently vanish from
	// the totals. It gets a warn log and a data_warnings entry, and the
	// briefing still returns.
	st := store.NewMockStore()
	// Inject into the very store the briefing will read — mutating a second
	// MockStore instance would assert nothing.
	for _, pd := range st.GetRaw() {
		pd.Steps = append(pd.Steps, types.Step{
			ID: "broken-stamp", Title: "Битая метка", Status: types.StepDone,
			ProjectID: pd.Project.ID,
			UpdatedAt: mustBroken("2026-13-45"),
		})
		break
	}

	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("a bad timestamp must not fail the briefing, got: %v", err)
	}
	if len(b.DataWarnings) == 0 {
		t.Fatal("data_warnings is empty — the broken stamp was swallowed silently")
	}
	joined := strings.Join(b.DataWarnings, "; ")
	if !strings.Contains(joined, "2026-13-45") {
		t.Errorf("warning must name the raw value, got: %s", joined)
	}
	if !strings.Contains(joined, "broken-stamp") {
		t.Errorf("warning must name the step, got: %s", joined)
	}
}

func TestGenerate_CleanStoreHasNoWarnings(t *testing.T) {
	st := store.NewMockStore()
	b, err := Generate(Config{Store: st, Date: "2026-06-14", Loc: time.UTC})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(b.DataWarnings) != 0 {
		t.Errorf("clean store produced warnings: %v", b.DataWarnings)
	}
}

func TestBriefing_WarningsReachTheWire(t *testing.T) {
	b := &Briefing{Date: "2026-06-14"}
	b.warn("step x", "updated_at", "2026-13-45")
	out, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	if !strings.Contains(string(out), "data_warnings") {
		t.Errorf("data_warnings missing from JSON: %s", out)
	}
	// And it must be absent when clean, not an empty array noise.
	clean, _ := json.Marshal(&Briefing{Date: "2026-06-14"})
	if strings.Contains(string(clean), "data_warnings") {
		t.Errorf("clean briefing should omit data_warnings: %s", clean)
	}
}

func TestGenerate_LastStepPicksTheNewestInstant(t *testing.T) {
	// String comparison used to decide this. Same calendar day, different
	// instants: the later one must win.
	steps := []types.Step{
		{ID: "morning", Status: types.StepDone, Title: "утро", UpdatedAt: mustTS("2026-06-14T09:00:00Z")},
		{ID: "evening", Status: types.StepDone, Title: "вечер", UpdatedAt: mustTS("2026-06-14T19:57:55Z")},
	}
	pd := types.ProjectData{Project: types.Project{ID: "p1", Title: "T"}, Steps: steps}
	ps := buildProjectSection(pd)
	if ps.LastStep != "вечер" {
		t.Errorf("LastStep = %q, want %q (newest instant wins)", ps.LastStep, "вечер")
	}
}

func TestBlockerDaysAlive_CountsCalendarDays(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, moscow)
	bl := types.Blocker{CreatedAt: mustTS("2026-09-20T23:00:00Z")} // 2026-09-21 02:00 MSK
	days, ok := blockerDaysAlive(bl, now, moscow)
	if !ok {
		t.Fatal("a valid stamp must not report invalid")
	}
	if days != 8 {
		t.Errorf("DaysAlive = %d, want 8 (2026-09-21 → 2026-09-29 local)", days)
	}
}

func TestBlockerDaysAlive_InvalidReportsInsteadOfZero(t *testing.T) {
	bl := types.Blocker{CreatedAt: mustBroken("nonsense")}
	days, ok := blockerDaysAlive(bl, time.Now(), time.UTC)
	if ok {
		t.Error("an unreadable stamp must report invalid, not a fabricated 0")
	}
	if days != 0 {
		t.Errorf("days = %d, want 0 alongside the invalid signal", days)
	}
}
