package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/xvantz/pm/internal/briefing"
	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/slug"
	"github.com/xvantz/pm/internal/types"
)

// RegisterPMTools registers all PM MCP tools on the server.
func RegisterPMTools(s *Server, st store.Store) {
	tools := []Tool{
		{
			Name:        "list_projects",
			Description: "List all projects with their status and progress",
			InputSchema: json.RawMessage(`{}`),
			Handler:     makeHandler(st, handleListProjects),
		},
		{
			Name:        "get_project",
			Description: "Get full project details including steps and decisions",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleGetProject),
		},
		{
			Name:        "add_project",
			Description: "Create a new project",
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
			Description: "Add a step to a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"},
					"title": {"type": "string", "description": "Step title"}
				},
				"required": ["project_id", "title"]
			}`),
			Handler: makeHandler(st, handleAddStep),
		},
		{
			Name:        "start_step",
			Description: "Mark a step as in_progress",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"},
					"step_id": {"type": "string", "description": "Step slug/ID"}
				},
				"required": ["project_id", "step_id"]
			}`),
			Handler: makeHandler(st, handleStartStep),
		},
		{
			Name:        "review_step",
			Description: "Send a step to review (agent completes, human approves)",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"},
					"step_id": {"type": "string", "description": "Step slug/ID"}
				},
				"required": ["project_id", "step_id"]
			}`),
			Handler: makeHandler(st, handleReviewStep),
		},
		{
			Name:        "done_step",
			Description: "Mark a step as done (must be in review first)",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"},
					"step_id": {"type": "string", "description": "Step slug/ID"}
				},
				"required": ["project_id", "step_id"]
			}`),
			Handler: makeHandler(st, handleDoneStep),
		},
		{
			Name:        "add_blocker",
			Description: "Add a blocker to a step",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"},
					"step_id": {"type": "string", "description": "Step slug/ID"},
					"title": {"type": "string", "description": "Blocker title"},
					"reason": {"type": "string", "description": "Why this blocker exists (optional)"}
				},
				"required": ["project_id", "step_id", "title"]
			}`),
			Handler: makeHandler(st, handleAddBlocker),
		},
		{
			Name:        "resolve_blocker",
			Description: "Resolve a blocker on a step",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"},
					"step_id": {"type": "string", "description": "Step slug/ID"},
					"blocker_id": {"type": "string", "description": "Blocker slug/ID"}
				},
				"required": ["project_id", "step_id", "blocker_id"]
			}`),
			Handler: makeHandler(st, handleResolveBlocker),
		},
		{
			Name:        "add_decision",
			Description: "Record an architectural or project decision",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"},
					"title": {"type": "string", "description": "Decision title"},
					"reason": {"type": "string", "description": "Rationale for the decision (optional)"}
				},
				"required": ["project_id", "title"]
			}`),
			Handler: makeHandler(st, handleAddDecision),
		},
		{
			Name:        "get_briefing",
			Description: "Generate a daily project briefing with recommendations",
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
			Description: "List all steps in a project with their status",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleListSteps),
		},
		{
			Name:        "list_blockers",
			Description: "List all blockers in a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleListBlockers),
		},
		{
			Name:        "list_decisions",
			Description: "List all decisions recorded in a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"project_id": {"type": "string", "description": "Project number or UUID"}
				},
				"required": ["project_id"]
			}`),
			Handler: makeHandler(st, handleListDecisions),
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
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
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

func handleGetProject(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(pd)
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}

func handleAddProject(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Title string   `json:"title"`
		Goal  string   `json:"goal,omitempty"`
		Tags  []string `json:"tags,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
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
	now := types.NowISO()
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

	if err := st.SaveProject(p); err != nil {
		return "", fmt.Errorf("save project: %w", err)
	}

	return fmt.Sprintf("Project #%d %q created.\nID: %s\n\nNext: pm add step %d \"...\"",
		p.Number, p.Title, p.ID, p.Number), nil
}

func handleAddStep(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
	}

	id := slug.Of(params.Title)
	if id == "" {
		return "", fmt.Errorf("invalid step title: %q", params.Title)
	}

	// Check for duplicate
	for _, s := range pd.Steps {
		if s.ID == id {
			return "", fmt.Errorf("step %q already exists in project #%d", id, pd.Project.Number)
		}
	}

	now := types.NowISO()
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

	return fmt.Sprintf("Step %q added to project #%d.\nStatus: todo\n\nNext: pm step start %d %s",
		id, pd.Project.Number, pd.Project.Number, id), nil
}

func handleStartStep(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	result, err := advanceStep(st, params.ProjectID, params.StepID, types.StepInProgress,
		func(s types.Step) error {
			if s.Status != types.StepTodo {
				return fmt.Errorf("step %q is %s, can only start from todo", params.StepID, s.Status)
			}
			return nil
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
		return "", fmt.Errorf("invalid args: %w", err)
	}

	result, err := advanceStep(st, params.ProjectID, params.StepID, types.StepReview,
		func(s types.Step) error {
			if s.Status != types.StepTodo && s.Status != types.StepInProgress {
				return fmt.Errorf("step %q is %s, can only review todo or in_progress steps", params.StepID, s.Status)
			}
			return nil
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
		return "", fmt.Errorf("invalid args: %w", err)
	}

	result, err := advanceStep(st, params.ProjectID, params.StepID, types.StepDone,
		func(s types.Step) error {
			if s.Status != types.StepReview {
				return fmt.Errorf("step %q is %s, must be in review before done", params.StepID, s.Status)
			}
			// Check for unresolved blockers
			for _, b := range s.Blockers {
				if b.Status == types.BlockerActive || b.Status == types.BlockerWaiting {
					return fmt.Errorf("step %q has unresolved blockers — resolve them first", params.StepID)
				}
			}
			return nil
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
		return "", err
	}

	for i, s := range pd.Steps {
		if s.ID == stepID {
			if err := validate(s); err != nil {
				return "", err
			}

			pd.Steps[i].Status = newStatus
			pd.Steps[i].UpdatedAt = types.NowISO()

			if err := st.SaveStep(pd.Steps[i]); err != nil {
				return "", fmt.Errorf("save step: %w", err)
			}

			pd.Project.UpdatedAt = types.NowISO()
			if err := st.SaveProject(pd.Project); err != nil {
				// Non-fatal
				slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
			}

			return fmt.Sprintf("Step %q → %s in project #%d.", stepID, newStatus, pd.Project.Number), nil
		}
	}

	return "", fmt.Errorf("step %q not found in project #%d", stepID, pd.Project.Number)
}

func handleAddBlocker(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
		Title     string `json:"title"`
		Reason    string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("step %q not found in project #%d", params.StepID, pd.Project.Number)
	}

	id := slug.Of(params.Title)
	if id == "" {
		return "", fmt.Errorf("invalid blocker title: %q", params.Title)
	}

	// Check duplicate
	for _, b := range targetStep.Blockers {
		if b.ID == id {
			return "", fmt.Errorf("blocker %q already exists in step %q", id, params.StepID)
		}
	}

	now := types.NowISO()
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

	return fmt.Sprintf("Blocker %q added to step %q in project #%d.",
		id, params.StepID, pd.Project.Number), nil
}

func handleResolveBlocker(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		StepID    string `json:"step_id"`
		BlockerID string `json:"blocker_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("step %q not found in project #%d", params.StepID, pd.Project.Number)
	}
	if blockerIdx == -1 {
		return "", fmt.Errorf("blocker %q not found in step %q", params.BlockerID, params.StepID)
	}

	blocker := &pd.Steps[stepIdx].Blockers[blockerIdx]
	if blocker.Status == types.BlockerResolved {
		return fmt.Sprintf("Blocker %q is already resolved in step %q (project #%d).",
			params.BlockerID, params.StepID, pd.Project.Number), nil
	}

	blocker.Status = types.BlockerResolved
	blocker.UpdatedAt = types.NowISO()

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

	pd.Project.UpdatedAt = types.NowISO()
	if err := st.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

	return fmt.Sprintf("Blocker %q resolved in step %q (project #%d).",
		params.BlockerID, params.StepID, pd.Project.Number), nil
}

func handleAddDecision(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
		Reason    string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
	}

	id := slug.Of(params.Title)
	if id == "" {
		return "", fmt.Errorf("invalid decision title: %q", params.Title)
	}

	// Check duplicate
	for _, d := range pd.Decisions {
		if d.ID == id {
			return "", fmt.Errorf("decision %q already exists in project #%d", id, pd.Project.Number)
		}
	}

	now := types.NowISO()
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

	return fmt.Sprintf("Decision %q recorded in project #%d.", id, pd.Project.Number), nil
}

func handleGetBriefing(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Date      string `json:"date,omitempty"`
		ProjectID string `json:"project_id,omitempty"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		slog.Warn("get_briefing: ignoring invalid params", "error", err)
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	cfg := briefing.Config{
		Context:       ctx,
		Store:         st,
		Date:          params.Date,
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

func handleListSteps(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
	}

	steps := make([]jsonStepItem, 0, len(pd.Steps))
	for _, s := range pd.Steps {
		blockers := make([]jsonBlockerItem, 0, len(s.Blockers))
		for _, bl := range s.Blockers {
			blockers = append(blockers, jsonBlockerItem{
				ID:     bl.ID,
				Title:  bl.Title,
				Status: string(bl.Status),
				Reason: bl.Reason,
			})
		}
		steps = append(steps, jsonStepItem{
			ID:       s.ID,
			Title:    s.Title,
			Status:   string(s.Status),
			Blockers: blockers,
		})
	}

	data, err := json.Marshal(map[string]any{
		"project_number": pd.Project.Number,
		"project_title":  pd.Project.Title,
		"steps":          steps,
	})
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}

func handleListBlockers(st store.Store, ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
	}

	groups := make([]jsonBlockerGroup, 0)
	for _, s := range pd.Steps {
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
		return "", fmt.Errorf("invalid args: %w", err)
	}

	pd, err := st.ResolveProject(params.ProjectID)
	if err != nil {
		return "", err
	}

	decisions := pd.Decisions
	items := make([]jsonDecisionItem, 0, len(decisions))
	for _, d := range decisions {
		items = append(items, jsonDecisionItem{
			ID:     d.ID,
			Title:  d.Title,
			Date:   d.Date,
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
