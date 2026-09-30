package briefing

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// dayLayout is the calendar-day key used for bucketing and for the `date`
// query parameter. It is a presentation format, not an event format: event
// times are types.Timestamp (RFC3339 UTC).
const dayLayout = "2006-01-02"

type Briefing struct {
	GeneratedAt string    `json:"generated_at"`
	Date        string    `json:"date"`
	Summary     Summary   `json:"summary"`
	Sections    []Section `json:"sections"`
	// DataWarnings lists every event time that could not be read. Non-empty
	// means the counts above are incomplete and the caller should say so
	// rather than present partial numbers as whole.
	DataWarnings    []string         `json:"data_warnings,omitempty"`
	Recommendations []Recommendation `json:"recommendations"`
}

type Summary struct {
	ActiveProjects    int `json:"active_projects"`
	BlockedProjects   int `json:"blocked_projects"`
	CompletedProjects int `json:"completed_projects"`
	IdeaProjects      int `json:"idea_projects"`
	PausedProjects    int `json:"paused_projects"`
	TotalProjects     int `json:"total_projects"`

	StepsToday    int `json:"steps_today"`
	StepsThisWeek int `json:"steps_this_week"`
	ProjectsMoved int `json:"projects_advanced"`

	LongLivedBlockers []BlockedItem `json:"long_lived_blockers,omitempty"`
}

type BlockedItem struct {
	ProjectID    string `json:"project_id"`
	ProjectTitle string `json:"project_title"`
	BlockerTitle string `json:"blocker_title"`
	Reason       string `json:"reason"`
	DaysAlive    int    `json:"days_alive"`
}

type Section struct {
	Title    string           `json:"title"`
	Type     string           `json:"type"`
	Projects []ProjectSection `json:"projects"`
}

type ProjectSection struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Goal           string   `json:"goal,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	StepsTotal     int      `json:"steps_total"`
	StepsDone      int      `json:"steps_done"`
	LastStep       string   `json:"last_step,omitempty"`
	NextStep       string   `json:"next_step,omitempty"`
	BlockersActive int      `json:"blockers_active"`
}

type Recommendation struct {
	ProjectID string `json:"project_id"`
	StepID    string `json:"step_id,omitempty"`
	Title     string `json:"title"`
	Reason    string `json:"reason"`
	Priority  int    `json:"priority"`
}

type Config struct {
	Context       context.Context // optional, checked before long operations
	Store         store.Store
	Date          string // calendar day to generate briefing for (YYYY-MM-DD, default: today)
	FilterProject string // project ref (number or UUID) for single-project briefing

	// Loc is the time zone used to decide which calendar day an event belongs
	// to. Nil means time.Local, which for the daemon is the host zone. The
	// single-writer daemon is the thing that defines "today" for every client,
	// so the day must not be decided in UTC: a step closed at 01:00 local time
	// is today's work, not yesterday's.
	Loc *time.Location
}

// loc resolves the zone once, so every count in one run agrees.
func (c Config) loc() *time.Location {
	if c.Loc != nil {
		return c.Loc
	}
	return time.Local
}

// warn records an unreadable event time. Loud by design: a zero time silently
// subtracted from a total is how a briefing ends up lying with a straight face.
func (b *Briefing) warn(entity, field, raw string) {
	slog.Warn("briefing: unparseable timestamp",
		"entity", entity, "field", field, "raw", raw)
	b.DataWarnings = append(b.DataWarnings,
		fmt.Sprintf("%s.%s: unparseable timestamp %q — counts exclude it", entity, field, raw))
}

// dayKey buckets an event into a calendar day in loc, reporting an unreadable
// stamp instead of treating it as the epoch.
func dayKey(ts types.Timestamp, loc *time.Location) (string, bool) {
	if _, invalid := ts.Invalid(); invalid {
		return "", false
	}
	if ts.IsZero() {
		return "", true
	}
	return ts.DayKey(loc), true
}

func Generate(cfg Config) (*Briefing, error) {
	loc := cfg.loc()
	date := cfg.Date
	if date == "" {
		// Today in the reader's zone, not in UTC: the day a user means is the
		// one their clock says.
		date = time.Now().In(loc).Format(dayLayout)
	}

	if cfg.Context != nil {
		select {
		case <-cfg.Context.Done():
			return nil, cfg.Context.Err()
		default:
		}
	}

	projects, err := cfg.Store.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	// Single-project filter
	if cfg.FilterProject != "" {
		pd, err := cfg.Store.ResolveProject(cfg.FilterProject)
		if err != nil {
			return nil, err
		}
		projects = []types.Project{pd.Project}
	}

	b := &Briefing{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Date:        date,
	}

	var activeSec, blockedSec, completedSec, ideaSec []ProjectSection
	todaySteps := 0
	weekSteps := 0
	projectsMoved := 0
	totalBlocked := 0
	longBlockers := []BlockedItem{}

	// The requested day is a calendar day in the reader's zone. Parsed leniently:
	// an unrecognised `date` argument falls back to today, as before.
	briefingDate, err := time.ParseInLocation(dayLayout, date, loc)
	if err != nil {
		briefingDate = time.Now().In(loc)
	}

	weekStart := briefingDate.AddDate(0, 0, -7)

	for _, p := range projects {
		pd, err := cfg.Store.GetProject(p.ID)
		if err != nil {
			continue
		}

		ps := buildProjectSection(*pd)
		stepsDoneToday := countStepsOnDay(pd.Steps, date, types.StepDone, loc, b)
		stepsDoneThisWeek := countStepsSince(pd.Steps, weekStart, types.StepDone, loc, b)
		todaySteps += stepsDoneToday
		weekSteps += stepsDoneThisWeek
		if stepsDoneToday > 0 {
			projectsMoved++
		}

		activeBlockers := countActiveBlockers(pd.Steps)
		ps.BlockersActive = activeBlockers

		switch p.Status {
		case types.StatusActive:
			if activeBlockers > 0 {
				totalBlocked++
				blockedSec = append(blockedSec, ps)

				for _, bl := range collectBlockers(pd.Steps) {
					if bl.Status == types.BlockerActive || bl.Status == types.BlockerWaiting {
						days, ok := blockerDaysAlive(bl, briefingDate, loc)
						if !ok {
							b.warn(fmt.Sprintf("blocker %s/%s", p.ID, bl.ID), "created_at", blockerRaw(bl))
							continue
						}
						if days > 7 {
							longBlockers = append(longBlockers, BlockedItem{
								ProjectID: p.ID, ProjectTitle: p.Title,
								BlockerTitle: bl.Title, Reason: bl.Reason, DaysAlive: days,
							})
						}
					}
				}
			} else {
				activeSec = append(activeSec, ps)
			}
		case types.StatusCompleted:
			completedSec = append(completedSec, ps)
		case types.StatusPaused:
			// not shown in sections, counted in summary
		case types.StatusIdea:
			ideaSec = append(ideaSec, ps)
		}
	}

	// Sort sections: most done first for active, newest first for completed
	sort.Slice(activeSec, func(i, j int) bool {
		return activeSec[i].StepsDone > activeSec[j].StepsDone
	})
	sort.Slice(blockedSec, func(i, j int) bool {
		return blockedSec[i].BlockersActive > blockedSec[j].BlockersActive
	})

	// Assemble sections in display order
	if len(activeSec) > 0 {
		b.Sections = append(b.Sections, Section{
			Title: fmt.Sprintf("Активные проекты (%d)", len(activeSec)),
			Type:  "active", Projects: activeSec,
		})
	}
	if len(blockedSec) > 0 {
		b.Sections = append(b.Sections, Section{
			Title: fmt.Sprintf("Заблокировано (%d)", len(blockedSec)),
			Type:  "blocked", Projects: blockedSec,
		})
	}
	if len(ideaSec) > 0 {
		b.Sections = append(b.Sections, Section{
			Title: fmt.Sprintf("Идеи (%d)", len(ideaSec)),
			Type:  "idea", Projects: ideaSec,
		})
	}
	if len(completedSec) > 0 {
		b.Sections = append(b.Sections, Section{
			Title: fmt.Sprintf("Завершено (%d)", len(completedSec)),
			Type:  "completed", Projects: completedSec,
		})
	}

	// Count statistics
	totalActive := 0
	totalCompleted := 0
	totalIdea := 0
	totalPaused := 0
	for _, p := range projects {
		switch p.Status {
		case types.StatusActive:
			totalActive++
		case types.StatusCompleted:
			totalCompleted++
		case types.StatusIdea:
			totalIdea++
		case types.StatusPaused:
			totalPaused++
		}
	}

	b.Summary = Summary{
		ActiveProjects:    totalActive,
		BlockedProjects:   totalBlocked,
		CompletedProjects: totalCompleted,
		IdeaProjects:      totalIdea,
		PausedProjects:    totalPaused,
		TotalProjects:     len(projects),
		StepsToday:        todaySteps,
		StepsThisWeek:     weekSteps,
		ProjectsMoved:     projectsMoved,
		LongLivedBlockers: longBlockers,
	}

	// Generate recommendations
	b.Recommendations = generateRecommendations(activeSec, blockedSec, projects, cfg)

	return b, nil
}

func buildProjectSection(pd types.ProjectData) ProjectSection {
	ps := ProjectSection{
		ID:    pd.Project.ID,
		Title: pd.Project.Title,
		Goal:  pd.Project.Goal,
		Tags:  pd.Project.Tags,
	}

	total := len(pd.Steps)
	done := 0
	var lastDone types.Step
	var lastDoneAt types.Timestamp
	for _, s := range pd.Steps {
		if s.Status == types.StepDone {
			done++
			// Compare as instants, not as strings. The old string comparison
			// only held while every value was a date; with RFC3339 it would
			// silently pick the wrong "last step".
			if _, invalid := s.UpdatedAt.Invalid(); !invalid && !s.UpdatedAt.IsZero() {
				if lastDoneAt.IsZero() || s.UpdatedAt.After(lastDoneAt) {
					lastDone = s
					lastDoneAt = s.UpdatedAt
				}
			}
		}
		if s.Status == types.StepTodo || s.Status == types.StepInProgress {
			if ps.NextStep == "" {
				ps.NextStep = s.Title
			}
		}
	}
	if lastDone.Title != "" {
		ps.LastStep = lastDone.Title
	}
	ps.StepsTotal = total
	ps.StepsDone = done

	return ps
}

// countStepsOnDay counts steps last touched on the given calendar day in loc.
//
// This replaces an `s.UpdatedAt == date` string comparison that could never
// match once timestamps became RFC3339 — it failed silently, reporting zero
// completed steps while looking perfectly healthy. Bucketing by day in the
// reader's zone is both correct and immune to the format change.
func countStepsOnDay(steps []types.Step, day string, status types.StepStatus, loc *time.Location, b *Briefing) int {
	count := 0
	for _, s := range steps {
		if s.Status != status {
			continue
		}
		key, ok := dayKey(s.UpdatedAt, loc)
		if !ok {
			b.warn(fmt.Sprintf("step %s", s.ID), "updated_at", rawOf(s.UpdatedAt))
			continue
		}
		if key == day {
			count++
		}
	}
	return count
}

// countStepsSince counts steps touched at or after `since` (an instant).
func countStepsSince(steps []types.Step, since time.Time, status types.StepStatus, loc *time.Location, b *Briefing) int {
	count := 0
	for _, s := range steps {
		if s.Status != status {
			continue
		}
		if raw, invalid := s.UpdatedAt.Invalid(); invalid {
			b.warn(fmt.Sprintf("step %s", s.ID), "updated_at", raw)
			continue
		}
		if s.UpdatedAt.IsZero() {
			continue
		}
		_ = loc
		if !s.UpdatedAt.Time.Before(since) {
			count++
		}
	}
	return count
}

func collectBlockers(steps []types.Step) []types.Blocker {
	var blockers []types.Blocker
	for _, st := range steps {
		blockers = append(blockers, st.Blockers...)
	}
	return blockers
}

func countActiveBlockers(steps []types.Step) int {
	count := 0
	for _, b := range collectBlockers(steps) {
		if b.Status == types.BlockerActive || b.Status == types.BlockerWaiting {
			count++
		}
	}
	return count
}

// blockerRaw returns the offending text of an unreadable stamp, for the warning.
func blockerRaw(b types.Blocker) string {
	raw, _ := b.CreatedAt.Invalid()
	return raw
}

func rawOf(ts types.Timestamp) string {
	raw, _ := ts.Invalid()
	return raw
}

// blockerDaysAlive reports the number of calendar days since the blocker was
// created, in loc.
//
// Both ends are truncated to local midnight first, so the answer is a
// calendar difference ("8 days since the 21st") rather than elapsed 24-hour
// steps. Elapsed steps drift: a blocker created at 02:00 on the 21st would
// report 9 days on the 29th at noon, which is a number nobody says out loud.
//
// The second return value is false when the timestamp is unreadable — the
// caller warns instead of reporting a fabricated zero.
func blockerDaysAlive(b types.Blocker, now time.Time, loc *time.Location) (int, bool) {
	if _, invalid := b.CreatedAt.Invalid(); invalid {
		return 0, false
	}
	if b.CreatedAt.IsZero() {
		return 0, true
	}
	// Both ends must be expressed in the SAME zone before taking calendar
	// fields: reading Year/Month/Day off a UTC instant while `now` is local
	// silently mixes the two calendars and drifts by a day near midnight.
	createdLocal := b.CreatedAt.Time.In(loc)
	todayLocal := now.In(loc)
	createdDay := time.Date(createdLocal.Year(), createdLocal.Month(), createdLocal.Day(),
		0, 0, 0, 0, loc)
	todayDay := time.Date(todayLocal.Year(), todayLocal.Month(), todayLocal.Day(),
		0, 0, 0, 0, loc)
	days := int(todayDay.Sub(createdDay).Hours() / 24)
	if days < 0 {
		return 0, true
	}
	return days, true
}

func generateRecommendations(active, blocked []ProjectSection, projects []types.Project, cfg Config) []Recommendation {
	var recs []Recommendation
	prio := 1

	// 1. Active unblocked projects with next steps
	for _, ps := range active {
		if ps.NextStep != "" {
			recs = append(recs, Recommendation{
				ProjectID: ps.ID, Title: ps.NextStep,
				Reason: fmt.Sprintf("Продолжить %s — осталось %d из %d шагов",
					strings.ToLower(ps.Title), ps.StepsTotal-ps.StepsDone, ps.StepsTotal),
				Priority: prio,
			})
			prio++
		}
	}

	// 2. Blocked projects — resolve blockers
	for _, ps := range blocked {
		if ps.BlockersActive > 0 {
			recs = append(recs, Recommendation{
				ProjectID: ps.ID,
				Title:     fmt.Sprintf("Разблокировать %s", ps.Title),
				Reason:    fmt.Sprintf("Заблокировано %d блокерами", ps.BlockersActive),
				Priority:  prio,
			})
			prio++
		}
	}

	return recs
}

// FormatMarkdown renders the briefing as human-readable markdown text.
// LLM-агент может использовать этот метод или читать JSON напрямую.
func (b *Briefing) FormatMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 📋 PM Briefing — %s\n\n", b.Date))

	// Summary
	s := b.Summary
	sb.WriteString("## Сводка\n\n")
	sb.WriteString(fmt.Sprintf("- Активных проектов: **%d**\n", s.ActiveProjects))
	if s.BlockedProjects > 0 {
		sb.WriteString(fmt.Sprintf("- Заблокировано: **%d**\n", s.BlockedProjects))
	}
	sb.WriteString(fmt.Sprintf("- Завершено: **%d**\n", s.CompletedProjects))
	sb.WriteString(fmt.Sprintf("- Идеи: **%d**\n", s.IdeaProjects))
	if s.PausedProjects > 0 {
		sb.WriteString(fmt.Sprintf("- Приостановлено: **%d**\n", s.PausedProjects))
	}
	sb.WriteString("\n")

	if s.StepsToday > 0 || s.StepsThisWeek > 0 {
		sb.WriteString("### Динамика\n\n")
		if s.StepsToday > 0 {
			sb.WriteString(fmt.Sprintf("- Сегодня завершено шагов: **%d**\n", s.StepsToday))
		}
		if s.StepsThisWeek > 0 {
			sb.WriteString(fmt.Sprintf("- За неделю завершено шагов: **%d**\n", s.StepsThisWeek))
		}
		if s.ProjectsMoved > 0 {
			sb.WriteString(fmt.Sprintf("- Продвинуто проектов: **%d**\n", s.ProjectsMoved))
		}
		sb.WriteString("\n")
	}

	if len(s.LongLivedBlockers) > 0 {
		sb.WriteString("### ⚠️ Долгоживущие блокеры\n\n")
		for _, bl := range s.LongLivedBlockers {
			sb.WriteString(fmt.Sprintf("- **%s / %s**: %d дней — %s\n",
				bl.ProjectTitle, bl.BlockerTitle, bl.DaysAlive, bl.Reason))
		}
		sb.WriteString("\n")
	}

	// Sections
	for _, sec := range b.Sections {
		sb.WriteString(fmt.Sprintf("## %s\n\n", sec.Title))
		for _, ps := range sec.Projects {
			progress := fmt.Sprintf("%d/%d", ps.StepsDone, ps.StepsTotal)
			tagStr := ""
			if len(ps.Tags) > 0 {
				tagStr = fmt.Sprintf(" `[%s]`", strings.Join(ps.Tags, ", "))
			}
			sb.WriteString(fmt.Sprintf("**%s**%s — %s шагов\n", ps.Title, tagStr, progress))
			if ps.Goal != "" {
				sb.WriteString(fmt.Sprintf("  > %s\n", ps.Goal))
			}
			if ps.BlockersActive > 0 {
				sb.WriteString(fmt.Sprintf("  🚫 Блокеров: %d\n", ps.BlockersActive))
			}
			if ps.NextStep != "" {
				sb.WriteString(fmt.Sprintf("  → Следующий шаг: %s\n", ps.NextStep))
			}
			sb.WriteString("\n")
		}
	}

	// Recommendations
	if len(b.Recommendations) > 0 {
		sb.WriteString("## 🎯 Рекомендации на сегодня\n\n")
		for i, rec := range b.Recommendations {
			sb.WriteString(fmt.Sprintf("%d. **%s** — %s\n", i+1, rec.Title, rec.Reason))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("_Сгенерировано: %s_\n", b.GeneratedAt))

	return sb.String()
}
