package briefing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

const dateFormat = "2006-01-02"

type Briefing struct {
	GeneratedAt string         `json:"generated_at"`
	Date        string         `json:"date"`
	Summary     Summary        `json:"summary"`
	Sections    []Section      `json:"sections"`
	Recommendations []Recommendation `json:"recommendations"`
}

type Summary struct {
	ActiveProjects   int            `json:"active_projects"`
	BlockedProjects  int            `json:"blocked_projects"`
	CompletedProjects int           `json:"completed_projects"`
	IdeaProjects     int            `json:"idea_projects"`
	PausedProjects   int            `json:"paused_projects"`
	TotalProjects    int            `json:"total_projects"`

	StepsToday      int `json:"steps_today"`
	StepsThisWeek   int `json:"steps_this_week"`
	ProjectsMoved   int `json:"projects_advanced"`

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
	Store         store.Store
	Date          string // ISO date to generate briefing for (default: today)
	FilterProject string // project ref (number or UUID) for single-project briefing
}

func Generate(cfg Config) (*Briefing, error) {
	date := cfg.Date
	if date == "" {
		date = time.Now().UTC().Format(dateFormat)
	}

	projects, err := cfg.Store.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	// Single-project filter
	if cfg.FilterProject != "" {
		pd, err := cfg.Store.ResolveProject(cfg.FilterProject)
		if err != nil || pd == nil {
			return nil, fmt.Errorf("project not found: %s", cfg.FilterProject)
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

	briefingDate := parseDate(date)
	if briefingDate.IsZero() {
		briefingDate = time.Now().UTC()
	}

	weekStart := briefingDate.AddDate(0, 0, -7)

	for _, p := range projects {
		pd, _ := cfg.Store.GetProject(p.ID)
		if pd == nil {
			continue
		}

		ps := buildProjectSection(*pd)
		stepsDoneToday := countStepsOnDate(pd.Steps, date, types.StepDone)
		stepsDoneThisWeek := countStepsSince(pd.Steps, weekStart, types.StepDone)
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
						days := blockerDaysAlive(bl, briefingDate)
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
	for _, s := range pd.Steps {
		if s.Status == types.StepDone {
			done++
			if lastDone.UpdatedAt == "" || s.UpdatedAt > lastDone.UpdatedAt {
				lastDone = s
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

func countStepsOnDate(steps []types.Step, date string, status types.StepStatus) int {
	count := 0
	for _, s := range steps {
		if s.Status == status && s.UpdatedAt == date {
			count++
		}
	}
	return count
}

func countStepsSince(steps []types.Step, since time.Time, status types.StepStatus) int {
	count := 0
	for _, s := range steps {
		if s.Status == status {
			t := parseDate(s.UpdatedAt)
			if !t.IsZero() && (t.Equal(since) || t.After(since)) {
				count++
			}
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

func blockerDaysAlive(b types.Blocker, now time.Time) int {
	if b.CreatedAt == "" {
		return 0
	}
	created := parseDate(b.CreatedAt)
	if created.IsZero() {
		return 0
	}
	days := int(now.Sub(created).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func parseDate(s string) time.Time {
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return time.Time{}
	}
	return t
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
