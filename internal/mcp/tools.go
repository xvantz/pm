package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/xvantz/pm/internal/briefing"
	"github.com/xvantz/pm/internal/domain"
	"github.com/xvantz/pm/internal/slug"
	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// badArgs keeps a malformed call actionable: it echoes what was wrong and
// which keys the tool needs, so the fix costs no discovery call.
func badArgs(err error, need string) error {
	return fmt.Errorf("invalid args: %v. Need: %s", err, need)
}

// RegisterPMTools registers all PM MCP tools on the server.
func RegisterPMTools(s *Server, st store.Store) {
	tools := []Tool{
		{
			Name:        "list_projects",
			Description: "List all projects with status and progress. Start here to find a project number, then read state with get_project.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			Handler:     makeHandler(st, handleListProjects),
		},
		{
			Name:        "get_project",
			Description: "Get a project summary: status, step counts by state, open blockers, last completed step. Use detail=true only when you need every step and decision. For one step use get_step — it is far cheaper than this tool with detail.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"detail": {"type": "boolean", "description": "Return the full project data with every step and decision instead of the summary (optional)"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleGetProject),
		},
		{
			Name:        "get_step",
			Description: "Get one step in full, with its blocker reasons and artifacts. Use this instead of reading the whole project when you only need one step; list_steps carries only blocker ids.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"step_id": {"type": "string", "description": "Step slug/ID (see id in list_steps)"}
				},
				"required": ["project_id", "step_id"]
			}`),
			Handler: makeHandler(st, handleGetStep),
		},
		{
			Name:        "add_project",
			Description: "Create a new project (status idea). Then add steps with add_step; to finish work use close_project, not delete_project.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "Project title"},
					"goal": {"type": "string", "description": "Project goal (optional)"},
					"tags": {"type": "array", "items": {"type": "string"}, "description": "Tags (optional)"}
				},
				"required": ["title"]
			}`),
			Handler: makeHandler(st, handleAddProject),
		},
		{
			Name:        "add_step",
			Description: "Add a step to a project (starts as todo). Advance it with start_step -> review_step -> done_step.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"title": {"type": "string", "description": "Step title"}
				},
				"required": ["project_id", "title"]
			}`),
			Handler: makeHandler(st, handleAddStep),
		},
		{
			Name:        "start_step",
			Description: "Begin work on a todo step. Next is review_step when work is done, never done_step directly.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"step_id": {"type": "string", "description": "Step slug/ID (see id in list_steps)"}
				},
				"required": ["project_id", "step_id"]
			}`),
			Handler: makeHandler(st, handleStartStep),
		},
		{
			Name:        "review_step",
			Description: "Mark work complete on an in_progress step; a human approves. Next is done_step. Blocked instead? Use add_blocker.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"step_id": {"type": "string", "description": "Step slug/ID (see id in list_steps)"}
				},
				"required": ["project_id", "step_id"]
			}`),
			Handler: makeHandler(st, handleReviewStep),
		},
		{
			Name:        "done_step",
			Description: "Mark a review step as done. Fails unless in review - call review_step first. To finish everything at once use close_project.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"step_id": {"type": "string", "description": "Step slug/ID (see id in list_steps)"}
				},
				"required": ["project_id", "step_id"]
			}`),
			Handler: makeHandler(st, handleDoneStep),
		},
		{
			Name:        "add_blocker",
			Description: "Flag what blocks a step; the step stays blocked until resolve_blocker clears it.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"step_id": {"type": "string", "description": "Step slug/ID (see id in list_steps)"},
					"title": {"type": "string", "description": "Blocker title"},
					"reason": {"type": "string", "description": "Why this blocker exists (optional)"}
				},
				"required": ["project_id", "step_id", "title"]
			}`),
			Handler: makeHandler(st, handleAddBlocker),
		},
		{
			Name:        "resolve_blocker",
			Description: "Clear a blocker by blocker_id (find ids in list_steps or get_step). The step keeps its status, it does not advance.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"step_id": {"type": "string", "description": "Step slug/ID (see id in list_steps)"},
					"blocker_id": {"type": "string", "description": "Blocker slug/ID (see blocker_ids in list_steps or blockers in get_step)"}
				},
				"required": ["project_id", "step_id", "blocker_id"]
			}`),
			Handler: makeHandler(st, handleResolveBlocker),
		},
		{
			Name:        "add_decision",
			Description: "Record why something was decided. Read back with list_decisions; close_project writes its own Closed decision.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"title": {"type": "string", "description": "Decision title"},
					"reason": {"type": "string", "description": "Rationale for the decision (optional)"}
				},
				"required": ["project_id", "title"]
			}`),
			Handler: makeHandler(st, handleAddDecision),
		},
		{
			Name:        "get_briefing",
			Description: "Daily briefing across projects: what changed, what is blocked, what to do next. For one project state use get_project instead.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"date": {"type": "string", "description": "ISO date (YYYY-MM-DD), defaults to today (optional)"},
					"project_id": {"type": "string", "description": "Filter to a single project (optional)"}
				}
			}`),
			Handler: makeHandler(st, handleGetBriefing),
		},
		{
			Name:        "list_steps",
			Description: "List a project's steps briefly: id, title, status, updated_at, blocker_ids. For one step's blocker reasons or artifacts use get_step.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleListSteps),
		},
		{
			Name:        "list_blockers",
			Description: "List unresolved blockers grouped by step, with reasons. Pass step_id for one step. Cheap counterpart: list_steps shows blocker_ids; full counterpart: get_step.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"step_id": {"type": "string", "description": "Step slug/ID to narrow to, see id in list_steps (optional)"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleListBlockers),
		},
		{
			Name:        "list_decisions",
			Description: "List a project's recorded decisions. To record one use add_decision.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleListDecisions),
		},
		{
			Name:        "close_project",
			Description: "Bulk-close a finished project in one call: force-completes all open steps, marks it completed, records the reason. Use instead of N start/review/done calls. REQUIRES CONSENT: call once without confirm to get a plan, show it to the human, then call again with confirm=true and a reason.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"},
					"reason": {"type": "string", "description": "Why it is closed, recorded as a decision. Required when confirm is true."},
					"confirm": {"type": "boolean", "description": "Consent flag. Omit or false to receive a plan without closing anything. Set true only after a human agreed to the plan."}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleCloseProject),
		},
		{
			Name:        "delete_project",
			Description: "Move a project to trash (recoverable via trash_restore). For finished work prefer close_project, which records why.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID (see list_projects)"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleDeleteProject),
		},
		{
			Name:        "trash_list",
			Description: "List trashed projects with names, numbers and deletion time. Restore with trash_restore; no wipe tool exists here on purpose, erasing is CLI-only.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
			Handler: makeHandler(st, handleTrashList),
		},
		{
			Name:        "trash_restore",
			Description: "Restore a trashed project by trash name, project number or title. Ambiguous matches fail with the candidate list instead of restoring; pass the exact trash name from trash_list to skip matching.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"target": {"type": "string", "description": "Trash name, project number or title from trash_list"}
				},
				"required": ["target"]
			}`),
			Handler: makeHandler(st, handleTrashRestore),
		},
	}

	for _, t := range tools {
		s.AddTool(t)
	}
}

// toolHandler is a function that processes a tool call.
type toolHandler func(st store.Store, ctx context.Context, args json.RawMessage) (string, error)

// makeHandler wraps a toolHandler into an MCP Tool handler.
func makeHandler(st store.Store, fn toolHandler) func(context.Context, json.RawMessage) (string, error) {
	return func(ctx context.Context, args json.RawMessage) (string, error) {
		return fn(st, ctx, args)
	}
}

// --- JSON response types for read handlers ---

type jsonProjectItem struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Tags      []string `json:"tags,omitempty"`
	Goal      string   `json:"goal,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type jsonBlockerItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type jsonStepItem struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Status   string            `json:"status"`
	Blockers []jsonBlockerItem `json:"blockers,omitempty"`
}

type jsonBlockerGroup struct {
	StepID    string            `json:"step_id"`
	StepTitle string            `json:"step_title"`
	Blockers  []jsonBlockerItem `json:"blockers"`
}

type jsonDecisionItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Date   string `json:"date"`
	Reason string `json:"reason,omitempty"`
}

// --- Handlers ---

func handleListProjects(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	projects, err := st.ListProjects()
	if err != nil {
		return "", fmt.Errorf("list projects: %w", err)
	}

	items := make([]jsonProjectItem, 0, len(projects))
	for _, p := range projects {
		items = append(items, jsonProjectItem{
			Number:    p.Number,
			Title:     p.Title,
			Status:    string(p.Status),
			Tags:      p.Tags,
			Goal:      p.Goal,
			CreatedAt: p.CreatedAt.String(),
			UpdatedAt: p.UpdatedAt.String(),
		})
	}

	data, err := json.Marshal(map[string]any{
		"count":    len(items),
		"projects": items,
	})
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}

// jsonStepDetail is the get_step answer: the full step plus a hint naming
// the next call for its state, so the detail level routes like the summary.
type jsonStepDetail struct {
	types.Step
	Hint string `json:"hint"`
}

// stepHint routes from a step's state: blocked steps name the clearing call,
// plain steps name the next lifecycle call.
func stepHint(projectRef string, s types.Step) string {
	for _, b := range s.Blockers {
		if b.Status == types.BlockerActive || b.Status == types.BlockerWaiting {
			return fmt.Sprintf("Step is blocked. Next: resolve_blocker {project_id: %q, step_id: %q, blocker_id: \"...\"} to clear; causes are in blockers above.", projectRef, s.ID)
		}
	}
	switch s.Status {
	case types.StepTodo:
		return fmt.Sprintf("Next: start_step {project_id: %q, step_id: %q} to begin work.", projectRef, s.ID)
	case types.StepInProgress:
		return fmt.Sprintf("Next: review_step {project_id: %q, step_id: %q} when work is done.", projectRef, s.ID)
	case types.StepReview:
		return fmt.Sprintf("Next: done_step {project_id: %q, step_id: %q} after human approval.", projectRef, s.ID)
	default:
		return fmt.Sprintf("Step is done. Next: get_project {project_id: %q} to see what remains.", projectRef)
	}
}

// jsonProjectSummary is the default get_project answer: enough to answer
// "where does this stand" without paying for every field of every step.
//
// Measured: a full ProjectData for a 7-step project is ~2.1 KB and a project
// dump cost more than listing every project. This shape is ~10x smaller and
// still carries what a caller reads first.
type jsonProjectSummary struct {
	Number         int            `json:"number"`
	Title          string         `json:"title"`
	Status         string         `json:"status"`
	Goal           string         `json:"goal,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	CompletedAt    string         `json:"completed_at,omitempty"`
	StepsTotal     int            `json:"steps_total"`
	StepsDone      int            `json:"steps_done"`
	StepsByStatus  map[string]int `json:"steps_by_status"`
	BlockersActive int            `json:"blockers_active"`
	LastStep       string         `json:"last_step,omitempty"`
	LastStepAt     string         `json:"last_step_at,omitempty"`
	Hint           string         `json:"hint"`
}

// buildProjectSummary counts by status instead of listing steps. Every
// non-done step shows up in the counts, so nothing is hidden — it is just not
// spelled out until someone asks for the step list.
func buildProjectSummary(pd types.ProjectData) jsonProjectSummary {
	s := jsonProjectSummary{
		Number:        pd.Project.Number,
		Title:         pd.Project.Title,
		Status:        string(pd.Project.Status),
		Goal:          pd.Project.Goal,
		Tags:          pd.Project.Tags,
		CreatedAt:     pd.Project.CreatedAt.String(),
		UpdatedAt:     pd.Project.UpdatedAt.String(),
		StepsByStatus: map[string]int{},
	}
	if !pd.Project.CompletedAt.IsZero() {
		s.CompletedAt = pd.Project.CompletedAt.String()
	}

	var lastDone types.Step
	var lastDoneAt types.Timestamp
	for _, st := range pd.Steps {
		s.StepsTotal++
		s.StepsByStatus[string(st.Status)]++
		if st.Status == types.StepDone {
			s.StepsDone++
			if _, invalid := st.UpdatedAt.Invalid(); !invalid && !st.UpdatedAt.IsZero() {
				if lastDoneAt.IsZero() || st.UpdatedAt.After(lastDoneAt) {
					lastDone, lastDoneAt = st, st.UpdatedAt
				}
			}
		}
		for _, b := range st.Blockers {
			if b.Status == types.BlockerActive || b.Status == types.BlockerWaiting {
				s.BlockersActive++
			}
		}
	}
	if lastDone.Title != "" {
		s.LastStep = lastDone.Title
		s.LastStepAt = lastDoneAt.String()
	}

	switch {
	case s.BlockersActive > 0:
		s.Hint = fmt.Sprintf("%d unresolved blocker(s). Next: list_blockers {project_id: %d} for reasons, resolve_blocker to clear.", s.BlockersActive, s.Number)
	case s.StepsDone < s.StepsTotal:
		s.Hint = fmt.Sprintf("%d of %d steps open. Next: list_steps {project_id: %d} to see which.", s.StepsTotal-s.StepsDone, s.StepsTotal, s.Number)
	default:
		s.Hint = fmt.Sprintf("All %d steps done. Next: close_project {project_id: %d, confirm: true, reason: \"...\"} when the human agrees.", s.StepsTotal, s.Number)
	}
	return s
}

func handleGetProject(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Detail    bool   `json:"detail,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id}")
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	var payload any
	if params.Detail {
		payload = pd
	} else {
		payload = buildProjectSummary(*pd)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}

// jsonStepBrief is one step in a list: enough to plan, not enough to dump.
// blocker_ids names the step's blockers without carrying them: the caller
// learns which get_step or resolve_blocker to call next.
type jsonStepBrief struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
	BlockerIDs []string `json:"blocker_ids"`
}

func handleListSteps(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id}")
	}
	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}
	items := make([]jsonStepBrief, 0, len(pd.Steps))
	for _, st := range pd.Steps {
		ids := make([]string, 0, len(st.Blockers))
		for _, b := range st.Blockers {
			ids = append(ids, b.ID)
		}
		items = append(items, jsonStepBrief{
			ID:         st.ID,
			Title:      st.Title,
			Status:     string(st.Status),
			UpdatedAt:  st.UpdatedAt.String(),
			BlockerIDs: ids,
		})
	}
	data, err := json.Marshal(map[string]any{"count": len(items), "steps": items})
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}

// handleGetStep answers "what is on this one step" without the project around
// it. Without this, a detail read on a 20-step project still costs all 20.
func handleGetStep(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, step_id}")
	}
	if params.StepID == "" {
		return "", fmt.Errorf("step_id is required. See list_steps {project_id: %q} for live ids", params.ProjectID)
	}
	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}
	for _, st := range pd.Steps {
		if st.ID != params.StepID {
			continue
		}
		data, err := json.Marshal(jsonStepDetail{Step: st, Hint: stepHint(params.ProjectID, st)})
		if err != nil {
			return "", fmt.Errorf("marshal response: %w", err)
		}
		return string(data), nil
	}
	// Name both, so a cross-project mistake is obvious from the error alone.
	return "", fmt.Errorf("step %q not found in project #%d. See list_steps {project_id: %q} for live ids", params.StepID, pd.Project.Number, params.ProjectID)
}

func handleAddProject(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Title string   `json:"title"`
		Goal  string   `json:"goal,omitempty"`
		Tags  []string `json:"tags,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{title}")
	}
	if params.Title == "" {
		return "", fmt.Errorf("title is required")
	}
	if slug.Of(params.Title) == "" {
		return "", fmt.Errorf("invalid title: %q", params.Title)
	}

	uid, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate project id: %w", err)
	}
	id := uid.String()
	now := types.NowTimestamp()
	nextNum, err := st.NextNumber()
	if err != nil {
		return "", fmt.Errorf("next number: %w", err)
	}

	p := types.Project{
		ID:        id,
		Number:    nextNum,
		Title:     params.Title,
		Goal:      params.Goal,
		Status:    types.StatusActive,
		Tags:      params.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Advance the counter BEFORE SaveProject so a crash between them
	// skips a number (gap) rather than reusing one (duplicate).
	if err := st.AdvanceNextNumber(); err != nil {
		return "", fmt.Errorf("advance next number: %w", err)
	}

	if err := st.SaveProject(p); err != nil {
		return "", fmt.Errorf("save project: %w", err)
	}

	return fmt.Sprintf("Project #%d %q created.\nID: %s\n\nNext: add_step {project_id: %d} to add the first step",
		p.Number, p.Title, p.ID, p.Number), nil
}

func handleAddStep(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, title}")
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	id := slug.Of(params.Title)
	if id == "" {
		return "", fmt.Errorf("invalid step title %q: it slugifies to empty; use letters or digits", params.Title)
	}

	// Check for duplicate
	for _, s := range pd.Steps {
		if s.ID == id {
			return "", fmt.Errorf("step %q already exists in project #%d; reuse it or pick another title", id, pd.Project.Number)
		}
	}

	now := types.NowTimestamp()
	step := types.Step{
		ID: id, Title: params.Title,
		Status: types.StepTodo, ProjectID: pd.Project.ID,
		CreatedAt: now, UpdatedAt: now,
	}

	if err := st.SaveStep(step); err != nil {
		return "", fmt.Errorf("save step: %w", err)
	}

	pd.Project.UpdatedAt = now
	if err := st.SaveProject(pd.Project); err != nil {
		// Non-fatal: step was saved
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

	return fmt.Sprintf("Step %q added to project #%d.\nStatus: todo\n\nNext: start_step {project_id: %d, step_id: %q} to begin work",
		id, pd.Project.Number, pd.Project.Number, id), nil
}

func handleStartStep(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, step_id}")
	}

	result, err := advanceStep(st, params.ProjectID, params.StepID, types.StepInProgress,
		func(s types.Step) error {
			return domain.ValidateStepStart(s)
		})
	if err != nil {
		return "", err
	}
	return result, nil
}

func handleReviewStep(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, step_id}")
	}

	result, err := advanceStep(st, params.ProjectID, params.StepID, types.StepReview,
		func(s types.Step) error {
			return domain.ValidateStepReview(s)
		})
	if err != nil {
		return "", err
	}
	return result, nil
}

func handleDoneStep(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, step_id}")
	}

	result, err := advanceStep(st, params.ProjectID, params.StepID, types.StepDone,
		func(s types.Step) error {
			return domain.ValidateStepDone(s)
		})
	if err != nil {
		return "", err
	}
	return result, nil
}

// advanceStep is a helper for start/review/done: finds the step, validates, updates.
func advanceStep(st store.Store, projectRef, stepID string, newStatus types.StepStatus, validate func(types.Step) error) (string, error) {
	pd, err := st.ResolveProject(projectRef)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	for i, s := range pd.Steps {
		if s.ID == stepID {
			if err := validate(s); err != nil {
				return "", fmt.Errorf("%w (fix: move through start_step -> review_step -> done_step in order, or close_project to finish all)", err)
			}

			pd.Steps[i].Status = newStatus
			pd.Steps[i].UpdatedAt = types.NowTimestamp()

			if err := st.SaveStep(pd.Steps[i]); err != nil {
				return "", fmt.Errorf("save step: %w", err)
			}

			pd.Project.UpdatedAt = types.NowTimestamp()
			if err := st.SaveProject(pd.Project); err != nil {
				// Non-fatal
				slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
			}

			next := map[types.StepStatus]string{
				types.StepInProgress: "review_step",
				types.StepReview:     "done_step",
			}[newStatus]
			msg := fmt.Sprintf("Step %q → %s in project #%d.", stepID, newStatus, pd.Project.Number)
			if next == "" {
				msg += fmt.Sprintf(" Next: get_project {project_id: %q} to see what remains.", projectRef)
			} else {
				msg += fmt.Sprintf(" Next: %s {project_id: %q, step_id: %q}.", next, projectRef, stepID)
			}
			return msg, nil
		}
	}

	return "", fmt.Errorf("step %q not found in project #%d. See list_steps {project_id: %q} for live ids", stepID, pd.Project.Number, projectRef)
}

func handleAddBlocker(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
		Title     string `json:"title"`
		Reason    string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, step_id, title}")
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	// Find the step
	var targetStep *types.Step
	for i, s := range pd.Steps {
		if s.ID == params.StepID {
			targetStep = &pd.Steps[i]
			break
		}
	}
	if targetStep == nil {
		return "", fmt.Errorf("step %q not found in project #%d. See list_steps {project_id: %q} for live ids", params.StepID, pd.Project.Number, params.ProjectID)
	}

	id := slug.Of(params.Title)
	if id == "" {
		return "", fmt.Errorf("invalid blocker title %q: it slugifies to empty; use letters or digits", params.Title)
	}

	// Check duplicate
	for _, b := range targetStep.Blockers {
		if b.ID == id {
			return "", fmt.Errorf("blocker %q already exists in step %q; reuse it or rename", id, params.StepID)
		}
	}

	now := types.NowTimestamp()
	blocker := types.Blocker{
		ID: id, Title: params.Title,
		Status:    types.BlockerWaiting,
		Reason:    params.Reason,
		ProjectID: pd.Project.ID,
		StepID:    params.StepID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := st.SaveBlocker(blocker); err != nil {
		return "", fmt.Errorf("save blocker: %w", err)
	}

	pd.Project.UpdatedAt = now
	if err := st.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

	return fmt.Sprintf("Blocker %q added to step %q in project #%d (step is blocked).\n\nNext: resolve_blocker {project_id: %q, step_id: %q, blocker_id: %q} when cleared.",
		id, params.StepID, pd.Project.Number, params.ProjectID, params.StepID, id), nil
}

func handleResolveBlocker(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
		BlockerID string `json:"blocker_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, step_id, blocker_id}")
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	stepIdx := -1
	blockerIdx := -1
	for i := range pd.Steps {
		if pd.Steps[i].ID == params.StepID {
			stepIdx = i
			for j := range pd.Steps[i].Blockers {
				if pd.Steps[i].Blockers[j].ID == params.BlockerID {
					blockerIdx = j
					break
				}
			}
			break
		}
	}
	if stepIdx == -1 {
		return "", fmt.Errorf("step %q not found in project #%d. See list_steps {project_id: %q} for live ids", params.StepID, pd.Project.Number, params.ProjectID)
	}
	if blockerIdx == -1 {
		return "", fmt.Errorf("blocker %q not found in step %q. See list_blockers {project_id: %q, step_id: %q} for live ids", params.BlockerID, params.StepID, params.ProjectID, params.StepID)
	}

	blocker := &pd.Steps[stepIdx].Blockers[blockerIdx]
	if blocker.Status == types.BlockerResolved {
		return fmt.Sprintf("Blocker %q is already resolved in step %q (project #%d, step is %s).\n\nNext: start_step {project_id: %q, step_id: %q} to move it forward.",
			params.BlockerID, params.StepID, pd.Project.Number, pd.Steps[stepIdx].Status, params.ProjectID, params.StepID), nil
	}

	blocker.Status = types.BlockerResolved
	blocker.UpdatedAt = types.NowTimestamp()

	if err := st.SaveBlocker(*blocker); err != nil {
		return "", fmt.Errorf("save blocker: %w", err)
	}

	// Unblock step if no more active blockers
	stillBlocked := false
	for _, b := range pd.Steps[stepIdx].Blockers {
		if b.Status == types.BlockerWaiting || b.Status == types.BlockerActive {
			stillBlocked = true
			break
		}
	}
	if !stillBlocked {
		pd.Steps[stepIdx].Status = types.StepTodo
		if err := st.SaveStep(pd.Steps[stepIdx]); err != nil {
			return "", fmt.Errorf("save step (unblock): %w", err)
		}
	}

	pd.Project.UpdatedAt = types.NowTimestamp()
	if err := st.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

	msg := fmt.Sprintf("Blocker %q resolved in step %q (project #%d).",
		params.BlockerID, params.StepID, pd.Project.Number)
	if stillBlocked {
		msg += fmt.Sprintf(" Other blockers remain: list_blockers {project_id: %q, step_id: %q}.", params.ProjectID, params.StepID)
	} else {
		msg += fmt.Sprintf(" Step is unblocked (todo). Next: start_step {project_id: %q, step_id: %q}.", params.ProjectID, params.StepID)
	}
	return msg, nil
}

func handleAddDecision(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
		Reason    string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, title}")
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	id := slug.Of(params.Title)
	if id == "" {
		return "", fmt.Errorf("invalid decision title %q: it slugifies to empty; use letters or digits", params.Title)
	}

	// Check duplicate
	for _, d := range pd.Decisions {
		if d.ID == id {
			return "", fmt.Errorf("decision %q already exists in project #%d; reuse it or rename", id, pd.Project.Number)
		}
	}

	now := types.NowTimestamp()
	dec := types.Decision{
		ID: id, Title: params.Title,
		Reason: params.Reason, Date: now,
		ProjectID: pd.Project.ID,
	}

	if err := st.SaveDecision(dec); err != nil {
		return "", fmt.Errorf("save decision: %w", err)
	}

	pd.Project.UpdatedAt = now
	if err := st.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

	return fmt.Sprintf("Decision %q recorded in project #%d. See list_decisions {project_id: %q}.", id, pd.Project.Number, params.ProjectID), nil
}

func handleGetBriefing(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Date      string `json:"date,omitempty"`
		ProjectID string `json:"project_id,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{date?, project_id?}")
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	cfg := briefing.Config{
		Context: ctx,
		Store:   st,
		Date:    params.Date,
	}
	if params.ProjectID != "" {
		cfg.FilterProject = params.ProjectID
	}

	b, err := briefing.Generate(cfg)
	if err != nil {
		return "", fmt.Errorf("generate briefing: %w", err)
	}

	return b.FormatMarkdown(), nil
}

func handleListBlockers(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, step_id?}")
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	groups := make([]jsonBlockerGroup, 0)
	foundStep := params.StepID == ""
	for _, s := range pd.Steps {
		if params.StepID != "" && s.ID != params.StepID {
			continue
		}
		foundStep = true
		if len(s.Blockers) > 0 {
			blockers := make([]jsonBlockerItem, 0, len(s.Blockers))
			for _, bl := range s.Blockers {
				blockers = append(blockers, jsonBlockerItem{
					ID:     bl.ID,
					Title:  bl.Title,
					Status: string(bl.Status),
					Reason: bl.Reason,
				})
			}
			groups = append(groups, jsonBlockerGroup{
				StepID:    s.ID,
				StepTitle: s.Title,
				Blockers:  blockers,
			})
		}
	}
	if params.StepID != "" && !foundStep {
		return "", fmt.Errorf("step %q not found in project #%d. See list_steps {project_id: %q} for live ids", params.StepID, pd.Project.Number, params.ProjectID)
	}

	data, err := json.Marshal(map[string]any{
		"project_number": pd.Project.Number,
		"project_title":  pd.Project.Title,
		"blockers":       groups,
	})
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}

func handleListDecisions(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id}")
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}

	decisions := pd.Decisions
	items := make([]jsonDecisionItem, 0, len(decisions))
	for _, d := range decisions {
		items = append(items, jsonDecisionItem{
			ID:     d.ID,
			Title:  d.Title,
			Date:   d.Date.String(),
			Reason: d.Reason,
		})
	}

	data, err := json.Marshal(map[string]any{
		"project_number": pd.Project.Number,
		"project_title":  pd.Project.Title,
		"decisions":      items,
	})
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}

func handleCloseProject(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Reason    string `json:"reason,omitempty"`
		Confirm   bool   `json:"confirm,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id, reason?, confirm?}")
	}
	if params.ProjectID == "" {
		return "", fmt.Errorf("project_id is required. See list_projects for numbers")
	}

	plan, err := st.CloseProject(params.ProjectID, params.Reason, params.Confirm)
	if err != nil {
		if strings.Contains(err.Error(), "reason is required") {
			return "", fmt.Errorf("%w. Call close_project again with confirm: true and reason after the human agrees to the plan", err)
		}
		return "", err
	}

	// No confirm: the store returned a plan and changed nothing. Render it as
	// something an agent can paste to a human, because the next step is that
	// a human decides.
	if !params.Confirm {
		return formatClosePlan(plan), nil
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}
	return fmt.Sprintf("Project #%d %q closed (%s). %d step(s) moved to done. Recorded as a Closed decision.",
		pd.Project.Number, pd.Project.Title, params.Reason, plan.StepsToClose()), nil
}

// formatClosePlan renders the preview for an agent to show a human.
//
// Blocker wording is deliberate: closing does not resolve blockers, the records
// stay on the completed steps. Saying otherwise would let someone approve a
// close believing a blocker had been settled.
func formatClosePlan(plan *types.ClosePlan) string {
	if plan == nil {
		return "Nothing to close."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Project %q is NOT closed yet — consent is required.\n\n", plan.ProjectTitle)

	if plan.StepsToClose() == 0 {
		sb.WriteString("All steps are already done. Closing would only mark the project completed.\n")
	} else {
		fmt.Fprintf(&sb, "Closing will mark %d step(s) done:\n", plan.StepsToClose())
		for _, s := range plan.Steps {
			fmt.Fprintf(&sb, "  - %s [%s]\n", s.Title, s.Status)
		}
	}

	if n := plan.BlockersToResolve(); n > 0 {
		fmt.Fprintf(&sb, "\n%d unresolved blocker(s) on these steps:\n", n)
		for _, b := range plan.Blockers {
			if b.Reason != "" {
				fmt.Fprintf(&sb, "  - %s (step %s): %s\n", b.Title, b.StepName, b.Reason)
			} else {
				fmt.Fprintf(&sb, "  - %s (step %s)\n", b.Title, b.StepName)
			}
		}
		sb.WriteString("NOTE: closing does NOT resolve blockers — these records stay on the completed steps.\n")
	}

	sb.WriteString("\nTo proceed, show this to the human and, if they agree, call again with:\n")
	fmt.Fprintf(&sb, "  confirm: true, reason: \"<why the project is closed>\"\n")
	sb.WriteString("A reason is required; it is recorded as the project decision.\n")
	return sb.String()
}

func handleDeleteProject(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{project_id}")
	}
	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("%w. See list_projects for live numbers", err)
	}
	if err := st.DeleteProject(pd.Project.ID); err != nil {
		return "", fmt.Errorf("delete project: %w", err)
	}
	return fmt.Sprintf("Project #%d %q moved to trash.\n\nUndo: trash_list to find it, trash_restore to bring it back.", pd.Project.Number, pd.Project.Title), nil
}

// handleTrashList renders the trash for an agent to act on: every entry
// carries the exact trash name the restore call needs, so there is no reason
// to guess.
func handleTrashList(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	items, err := st.TrashList()
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "Trash is empty.", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d trashed project(s). Restore with trash_restore and the trash name.\n", len(items))
	for _, it := range items {
		when := ""
		if !it.DeletedAt.IsZero() {
			when = it.DeletedAt.String()
		}
		fmt.Fprintf(&sb, "  - #%d %q (trash: %s, deleted %s)\n", it.Number, it.Title, it.TrashName, when)
	}
	return sb.String(), nil
}

// handleTrashRestore passes the target straight to the store, which resolves
// exact names, numbers and titles itself and fails loudly on ambiguity.
// The store is the single place that knows the matching rules.
func handleTrashRestore(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", badArgs(err, "{target}")
	}
	if strings.TrimSpace(params.Target) == "" {
		return "", fmt.Errorf("target is required: pass a trash name, project number or title from trash_list")
	}
	if err := st.TrashRestore(params.Target); err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Restored %q from trash.", params.Target)
	if pd, err := st.ResolveProject(params.Target); err == nil {
		msg += fmt.Sprintf(" Next: get_project {project_id: %d} to verify.", pd.Project.Number)
	} else {
		msg += " Verify with list_projects, then get_project."
	}
	return msg, nil
}
