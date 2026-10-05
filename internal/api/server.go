// Package api implements `pm serve`: a single-writer HTTP daemon owning the
// YAML store. Clients (CLI, MCP, future) talk to it instead of touching
// files directly, so concurrent writes serialize and the data path is known
// in exactly one place (the daemon).
package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/xvantz/pm/internal/briefing"
	"github.com/xvantz/pm/internal/domain"
	"github.com/xvantz/pm/internal/gitbackup"
	"github.com/xvantz/pm/internal/slug"
	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// Version is set by -ldflags during build; fallback for dev.
var Version = "dev"

// Server owns the store and serves it over HTTP.
type Server struct {
	store   store.Store
	mux     *http.ServeMux
	mu      sync.Mutex // serializes mutations across projects (counter, etc.)
	token   string
	version string
	backup  *gitbackup.Backup
}

// New returns a Server bound to st. Token must be non-empty.
func New(st store.Store, token string) *Server {
	return NewWithBackup(st, token, nil)
}

// NewWithBackup binds st with a git backup of the data dir. A nil or
// disabled backup is a no-op: handlers never branch on it.
func NewWithBackup(st store.Store, token string, bk *gitbackup.Backup) *Server {
	s := &Server{store: st, mux: http.NewServeMux(), token: token, version: Version, backup: bk}
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/projects", s.auth(s.handleProjectsList))
	s.mux.HandleFunc("POST /api/projects", s.auth(s.handleProjectCreate))
	s.mux.HandleFunc("GET /api/projects/{ref}", s.auth(s.handleProjectGet))
	s.mux.HandleFunc("PATCH /api/projects/{ref}", s.auth(s.handleProjectPatch))
	s.mux.HandleFunc("DELETE /api/projects/{ref}", s.auth(s.handleProjectDelete))
	s.mux.HandleFunc("POST /api/projects/{ref}/close", s.auth(s.handleProjectClose))
	s.mux.HandleFunc("GET /api/projects/{ref}/close-plan", s.auth(s.handleProjectClosePlan))
	s.mux.HandleFunc("GET /api/projects/{ref}/steps", s.auth(s.handleStepsList))
	s.mux.HandleFunc("POST /api/projects/{ref}/steps", s.auth(s.handleStepCreate))
	s.mux.HandleFunc("POST /api/projects/{ref}/steps/{step}/{action}", s.auth(s.handleStepAction))
	s.mux.HandleFunc("DELETE /api/projects/{ref}/steps/{step}", s.auth(s.handleStepDelete))
	s.mux.HandleFunc("GET /api/projects/{ref}/blockers", s.auth(s.handleBlockersList))
	s.mux.HandleFunc("POST /api/projects/{ref}/steps/{step}/blockers", s.auth(s.handleBlockerCreate))
	s.mux.HandleFunc("POST /api/projects/{ref}/steps/{step}/blockers/{blk}/resolve", s.auth(s.handleBlockerResolve))
	s.mux.HandleFunc("DELETE /api/projects/{ref}/steps/{step}/blockers/{blk}", s.auth(s.handleBlockerDelete))
	s.mux.HandleFunc("GET /api/projects/{ref}/decisions", s.auth(s.handleDecisionsList))
	s.mux.HandleFunc("POST /api/projects/{ref}/decisions", s.auth(s.handleDecisionCreate))
	s.mux.HandleFunc("DELETE /api/projects/{ref}/decisions/{dec}", s.auth(s.handleDecisionDelete))
	s.mux.HandleFunc("GET /api/briefing", s.auth(s.handleBriefing))
	s.mux.HandleFunc("GET /api/trash", s.auth(s.handleTrashList))
	s.mux.HandleFunc("POST /api/trash/{name}/restore", s.auth(s.handleTrashRestore))
	s.mux.HandleFunc("GET /api/doctor", s.auth(s.handleDoctor))
	s.mux.HandleFunc("DELETE /api/trash", s.auth(s.handleTrashClean))
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// auth rejects requests without the Bearer token.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("bad body: %v", err))
		return false
	}
	return true
}

// commit records the action in the data-dir git repo. Git failures are
// warnings: the data write already landed and the API answer stays green.
func (s *Server) commit(action string) {
	if err := s.backup.Commit(action); err != nil {
		slog.Warn("backup commit failed", "action", action, "error", err)
	}
}

// resolve finds the project or writes 404.
func (s *Server) resolve(w http.ResponseWriter, ref string) *types.ProjectData {
	pd, err := s.store.ResolveProject(ref)
	if err != nil {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("project %q not found", ref))
		return nil
	}
	return pd
}

func touchProject(s *Server, pd *types.ProjectData) {
	pd.Project.UpdatedAt = types.NowTimestamp()
	if err := s.store.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": s.version})
}

// --- projects ---

func (s *Server) handleProjectsList(w http.ResponseWriter, _ *http.Request) {
	projects, err := s.store.ListProjects()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []types.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

type createProjectReq struct {
	Title string   `json:"title"`
	Goal  string   `json:"goal,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	ID    string   `json:"id,omitempty"`
}

func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request) {
	var req createProjectReq
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, http.StatusBadRequest, "title cannot be empty")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// The store assigns the number and returns what was stored: one call,
	// no advisory read on this side. Errors map by sentinel.
	p, err := s.store.CreateProject(req.Title, req.Goal, req.Tags, req.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrEmptyTitle), errors.Is(err, store.ErrBadID):
			writeErr(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, store.ErrProjectExist):
			writeErr(w, http.StatusConflict, err.Error())
		default:
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("create project: %v", err))
		}
		return
	}
	writeJSON(w, http.StatusCreated, p)
	s.commit("add_project " + p.ID)
}

func (s *Server) handleProjectGet(w http.ResponseWriter, r *http.Request) {
	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	writeJSON(w, http.StatusOK, pd)
}

type patchProjectReq struct {
	Goal   *string              `json:"goal,omitempty"`
	Status *types.ProjectStatus `json:"status,omitempty"`
	Tags   []string             `json:"tags,omitempty"`
}

func (s *Server) handleProjectPatch(w http.ResponseWriter, r *http.Request) {
	var req patchProjectReq
	if !decode(w, r, &req) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	if req.Goal != nil {
		pd.Project.Goal = *req.Goal
	}
	if req.Status != nil {
		switch *req.Status {
		case types.StatusActive, types.StatusCompleted, types.StatusPaused, types.StatusIdea:
			pd.Project.Status = *req.Status
			if *req.Status == types.StatusCompleted {
				pd.Project.CompletedAt = types.NowTimestamp()
			}
		default:
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("bad status: %q", *req.Status))
			return
		}
	}
	if req.Tags != nil {
		pd.Project.Tags = req.Tags
	}
	pd.Project.UpdatedAt = types.NowTimestamp()
	if err := s.store.SaveProject(pd.Project); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("save project: %v", err))
		return
	}
	s.commit("patch_project " + pd.Project.ID)
	writeJSON(w, http.StatusOK, pd.Project)
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	if err := s.store.DeleteProject(pd.Project.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.commit("delete_project " + pd.Project.ID)
	writeJSON(w, http.StatusOK, map[string]string{"trashed": pd.Project.Title})
}

type closeProjectReq struct {
	Reason string `json:"reason,omitempty"`
	// Confirm gates the irreversible half. Without it the call returns a plan
	// and changes nothing: bulk close bypasses the step lifecycle on purpose,
	// so it is the one operation that can mark unfinished work finished.
	Confirm bool `json:"confirm,omitempty"`
}

// closeProjectResp carries a plan, a closed project, or both. Confirmed
// says which: false is the no-consent preview (plan only, nothing touched),
// true is the close itself (project plus the plan built before closing, so
// every caller can report the moved count from one response).
type closeProjectResp struct {
	Confirmed bool             `json:"confirmed"`
	Plan      *types.ClosePlan `json:"plan,omitempty"`
	Project   *types.Project   `json:"project,omitempty"`
}

func (s *Server) handleProjectClose(w http.ResponseWriter, r *http.Request) {
	var req closeProjectReq
	if r.ContentLength != 0 {
		if !decode(w, r, &req) {
			return
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}

	if !req.Confirm {
		// Preview only. Read-only path: nothing below this point may save.
		plan, err := s.store.ClosePlan(pd.Project.ID)
		if err != nil {
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, closeProjectResp{Confirmed: false, Plan: plan})
		return
	}

	// Consent given. The store enforces the reason and performs the close;
	// the daemon only shapes the response. The plan is built before closing
	// so confirm answers with both: file-backed and daemon-backed callers
	// then observe the same contract, and no caller dereferences a nil plan.
	plan, err := s.store.ClosePlan(pd.Project.ID)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, err := s.store.CloseProject(pd.Project.ID, req.Reason, true); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	closed, err := s.store.GetProject(pd.Project.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.commit("close_project " + pd.Project.ID)
	writeJSON(w, http.StatusOK, closeProjectResp{Confirmed: true, Project: &closed.Project, Plan: plan})
}

// handleProjectClosePlan answers "what would closing do" without closing.
// The two-call consent flow needs the plan available on its own too, so an
// agent can look before it even decides to ask, and the CLI can print the same
// list a human would see.
func (s *Server) handleProjectClosePlan(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	plan, err := s.store.ClosePlan(pd.Project.ID)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// --- steps ---

func (s *Server) handleStepsList(w http.ResponseWriter, r *http.Request) {
	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	steps, err := s.store.GetSteps(pd.Project.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if steps == nil {
		steps = []types.Step{}
	}
	writeJSON(w, http.StatusOK, steps)
}

type createStepReq struct {
	Title string `json:"title"`
}

func (s *Server) handleStepCreate(w http.ResponseWriter, r *http.Request) {
	var req createStepReq
	if !decode(w, r, &req) {
		return
	}
	id := slug.Of(req.Title)
	if id == "" {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid step title: %q", req.Title))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	for _, st := range pd.Steps {
		if st.ID == id {
			writeErr(w, http.StatusConflict, fmt.Sprintf("step %q already exists", id))
			return
		}
	}
	now := types.NowTimestamp()
	step := types.Step{
		ID: id, Title: req.Title, Status: types.StepTodo,
		ProjectID: pd.Project.ID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.SaveStep(step); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("save step: %v", err))
		return
	}
	touchProject(s, pd)
	s.commit("add_step " + pd.Project.ID + " " + step.ID)
	writeJSON(w, http.StatusCreated, step)
}

func (s *Server) findStep(pd *types.ProjectData, stepID string) *types.Step {
	for i := range pd.Steps {
		if pd.Steps[i].ID == stepID {
			return &pd.Steps[i]
		}
	}
	return nil
}

func (s *Server) handleStepAction(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	var validate func(types.Step) error
	var next types.StepStatus
	switch action {
	case "start":
		validate, next = domain.ValidateStepStart, types.StepInProgress
	case "review":
		validate, next = domain.ValidateStepReview, types.StepReview
	case "done":
		validate, next = domain.ValidateStepDone, types.StepDone
	default:
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("bad action: %q (start|review|done)", action))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	st := s.findStep(pd, r.PathValue("step"))
	if st == nil {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("step %q not found", r.PathValue("step")))
		return
	}
	if err := validate(*st); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	domain.StepStatusChange(st, next, types.NowTimestamp())
	if err := s.store.SaveStep(*st); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("save step: %v", err))
		return
	}
	touchProject(s, pd)
	s.commit(action + "_step " + pd.Project.ID + " " + st.ID)
	writeJSON(w, http.StatusOK, *st)
}

func (s *Server) handleStepDelete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	if s.findStep(pd, r.PathValue("step")) == nil {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("step %q not found", r.PathValue("step")))
		return
	}
	if err := s.store.DeleteStep(pd.Project.ID, r.PathValue("step")); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	touchProject(s, pd)
	s.commit("delete_step " + pd.Project.ID + " " + r.PathValue("step"))
	writeJSON(w, http.StatusOK, map[string]string{"deleted": r.PathValue("step")})
}

// --- blockers ---

func (s *Server) handleBlockersList(w http.ResponseWriter, r *http.Request) {
	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	blockers, err := s.store.GetBlockers(pd.Project.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if blockers == nil {
		blockers = []types.Blocker{}
	}
	writeJSON(w, http.StatusOK, blockers)
}

type createBlockerReq struct {
	Title  string `json:"title"`
	Reason string `json:"reason,omitempty"`
}

func (s *Server) handleBlockerCreate(w http.ResponseWriter, r *http.Request) {
	var req createBlockerReq
	if !decode(w, r, &req) {
		return
	}
	id := slug.Of(req.Title)
	if id == "" {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid blocker title: %q", req.Title))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	st := s.findStep(pd, r.PathValue("step"))
	if st == nil {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("step %q not found", r.PathValue("step")))
		return
	}
	for _, b := range st.Blockers {
		if b.ID == id {
			writeErr(w, http.StatusConflict, fmt.Sprintf("blocker %q already exists", id))
			return
		}
	}
	now := types.NowTimestamp()
	blocker := types.Blocker{
		ID: id, Title: req.Title, Status: types.BlockerWaiting,
		Reason: req.Reason, ProjectID: pd.Project.ID,
		StepID: st.ID, CreatedAt: now, UpdatedAt: now,
	}
	// SaveBlocker sets the step status to blocked and saves it.
	if err := s.store.SaveBlocker(blocker); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("save blocker: %v", err))
		return
	}
	// Re-read for the response (store applied step-status invariants).
	fresh, err := s.store.ResolveProject(pd.Project.ID)
	if err == nil {
		touchProject(s, fresh)
	} else {
		touchProject(s, pd)
	}
	s.commit("add_blocker " + pd.Project.ID + " " + st.ID + " " + blocker.ID)
	writeJSON(w, http.StatusCreated, blocker)
}

func (s *Server) handleBlockerResolve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	st := s.findStep(pd, r.PathValue("step"))
	if st == nil {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("step %q not found", r.PathValue("step")))
		return
	}
	var target *types.Blocker
	for i := range st.Blockers {
		if st.Blockers[i].ID == r.PathValue("blk") {
			target = &st.Blockers[i]
			break
		}
	}
	if target == nil {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("blocker %q not found", r.PathValue("blk")))
		return
	}
	target.Status = "resolved"
	target.UpdatedAt = types.NowTimestamp()
	// SaveBlocker applies step-blocked invariant but does NOT unblock:
	// unblocking lives here (same rule as `pm blocker resolve`).
	if err := s.store.SaveBlocker(*target); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("save blocker: %v", err))
		return
	}
	freshSteps, err := s.store.GetSteps(pd.Project.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, st := range freshSteps {
		if st.ID == r.PathValue("step") && !domain.HasUnresolvedBlockers(st.Blockers) {
			st.Status = types.StepTodo
			if err := s.store.SaveStep(st); err != nil {
				writeErr(w, http.StatusInternalServerError, fmt.Sprintf("unblock step: %v", err))
				return
			}
		}
	}
	touchProject(s, pd)
	s.commit("resolve_blocker " + pd.Project.ID + " " + r.PathValue("step") + " " + target.ID)
	writeJSON(w, http.StatusOK, *target)
}

func (s *Server) handleBlockerDelete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	if s.findStep(pd, r.PathValue("step")) == nil {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("step %q not found", r.PathValue("step")))
		return
	}
	if err := s.store.DeleteBlocker(pd.Project.ID, r.PathValue("step"), r.PathValue("blk")); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	touchProject(s, pd)
	s.commit("delete_blocker " + pd.Project.ID + " " + r.PathValue("step") + " " + r.PathValue("blk"))
	writeJSON(w, http.StatusOK, map[string]string{"deleted": r.PathValue("blk")})
}

// --- decisions ---

func (s *Server) handleDecisionsList(w http.ResponseWriter, r *http.Request) {
	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	decisions, err := s.store.GetDecisions(pd.Project.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if decisions == nil {
		decisions = []types.Decision{}
	}
	writeJSON(w, http.StatusOK, decisions)
}

type createDecisionReq struct {
	Title  string `json:"title"`
	Reason string `json:"reason,omitempty"`
}

func (s *Server) handleDecisionCreate(w http.ResponseWriter, r *http.Request) {
	var req createDecisionReq
	if !decode(w, r, &req) {
		return
	}
	id := slug.Of(req.Title)
	if id == "" {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid decision title: %q", req.Title))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	for _, d := range pd.Decisions {
		if d.ID == id {
			writeErr(w, http.StatusConflict, fmt.Sprintf("decision %q already exists", id))
			return
		}
	}
	now := types.NowTimestamp()
	decision := types.Decision{
		ID: id, Title: req.Title, Reason: req.Reason,
		Date: now, ProjectID: pd.Project.ID,
	}
	if err := s.store.SaveDecision(decision); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("save decision: %v", err))
		return
	}
	touchProject(s, pd)
	s.commit("add_decision " + pd.Project.ID + " " + decision.ID)
	writeJSON(w, http.StatusCreated, decision)
}

func (s *Server) handleDecisionDelete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pd := s.resolve(w, r.PathValue("ref"))
	if pd == nil {
		return
	}
	if err := s.store.DeleteDecision(pd.Project.ID, r.PathValue("dec")); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	touchProject(s, pd)
	s.commit("delete_decision " + pd.Project.ID + " " + r.PathValue("dec"))
	writeJSON(w, http.StatusOK, map[string]string{"deleted": r.PathValue("dec")})
}

// --- briefing ---

func (s *Server) handleBriefing(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	b, err := briefing.Generate(briefing.Config{
		Context:       r.Context(),
		Store:         s.store,
		Date:          q.Get("date"),
		FilterProject: q.Get("project"),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// --- trash ---

func (s *Server) handleTrashList(w http.ResponseWriter, _ *http.Request) {
	names, err := s.store.TrashList()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if names == nil {
		names = []types.TrashItem{}
	}
	writeJSON(w, http.StatusOK, names)
}

func (s *Server) handleTrashRestore(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := r.PathValue("name")
	if err := s.store.TrashRestore(name); err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	s.commit("trash_restore " + name)
	writeJSON(w, http.StatusOK, map[string]string{"restored": name})
}

func (s *Server) handleTrashClean(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.store.TrashClean(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.commit("trash_clean")
	writeJSON(w, http.StatusOK, map[string]string{"cleaned": "trash"})
}

// handleDoctor answers the integrity verdict over the daemon's own store.
// The CLI prints it instead of walking files itself: one reader means no
// divergence between what the human checks and what the daemon serves.
func (s *Server) handleDoctor(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rep, err := s.store.Check()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
