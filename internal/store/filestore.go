package store

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/xvantz/pm/internal/domain"
	"github.com/xvantz/pm/internal/types"
)

// FileStore reads/writes project data from YAML files on disk.
// All write operations use POSIX file locks (flock) on the project directory
// to coordinate between concurrent CLI and MCP processes. Writes are atomic:
// data is written to a temp file, synced to disk, then renamed into place.
type FileStore struct {
	root string // e.g. ./pm/projects
}

const (
	metaDir     = "_meta"
	nextNumFile = "next_number"
	backupsDir  = "backups"
)

// maxBackupRuns bounds _meta/backups: each deletion starts one run dir, and
// only the newest runs are kept. Twenty runs cover a long mistake tail
// without letting the store grow without bound.
const maxBackupRuns = 20

// backupRunDir ensures _meta/backups/<unixnano>/ exists and returns it.
// Nanosecond resolution keeps rapid successive deletes in separate runs so
// rotation actually counts deletions, not wall-clock seconds.
func (s *FileStore) backupRunDir() (string, error) {
	dir := filepath.Join(s.root, metaDir, backupsDir,
		strconv.FormatInt(time.Now().UnixNano(), 10))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create backup run dir: %w", err)
	}
	return dir, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// backupProjectDir copies a whole live project tree into the backup run.
// It copies what is being removed byte-for-byte, so a later human can put
// the files back by hand.
func (s *FileStore) backupProjectDir(id, runDir string) error {
	src := s.projectDir(id)
	dst := filepath.Join(runDir, id)
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		return copyFile(path, filepath.Join(dst, rel))
	})
}

// pruneBackups drops all but the newest maxBackupRuns runs. Best effort: a
// rotation failure must not fail the delete it trails, so errors are logged
// and swallowed.
func (s *FileStore) pruneBackups() {
	root := filepath.Join(s.root, metaDir, backupsDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	if len(entries) <= maxBackupRuns {
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names[:len(names)-maxBackupRuns] {
		if err := os.RemoveAll(filepath.Join(root, n)); err != nil {
			slog.Warn("prune backups", "run", n, "error", err)
		}
	}
}

// backupBeforeDelete snapshots one file about to be removed or rewritten.
// A backup WRITE failure fails the delete: if the disk cannot take the copy,
// proceeding would destroy data with no way back.
func (s *FileStore) backupBeforeDelete(projectID, relPath string) error {
	runDir, err := s.backupRunDir()
	if err != nil {
		return err
	}
	src := filepath.Join(s.projectDir(projectID), relPath)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("backup source %s: %w", relPath, err)
	}
	if err := copyFile(src, filepath.Join(runDir, projectID, relPath)); err != nil {
		return fmt.Errorf("backup %s: %w", relPath, err)
	}
	return nil
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

func (s *FileStore) ListProjects() ([]types.Project, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("read projects root %s: %w", s.root, err)
	}

	var projects []types.Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := s.readProject(e.Name())
		if err != nil {
			slog.Warn("skipping unreadable project", "dir", e.Name(), "error", err)
			continue
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

func (s *FileStore) GetProject(id string) (*types.ProjectData, error) {
	project, err := s.readProject(id)
	if err != nil {
		return nil, err
	}
	return s.loadProjectData(*project)
}

func (s *FileStore) ResolveProject(ref string) (*types.ProjectData, error) {
	// Try as number first — requires scanning all projects
	if n, err := strconv.Atoi(ref); err == nil {
		projects, err := s.ListProjects()
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.Number == n {
				return s.loadProjectData(p)
			}
		}
		return nil, fmt.Errorf("project #%d not found", n)
	}

	// Try as UUID (exact or prefix) — single scan
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	var matches []types.Project
	for _, p := range projects {
		if p.ID == ref {
			return s.loadProjectData(p)
		}
		if len(p.ID) >= len(ref) && p.ID[:len(ref)] == ref {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("project %q not found", ref)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous project prefix %q matches %d projects", ref, len(matches))
	}
	return s.loadProjectData(matches[0])
}

func (s *FileStore) NextNumber() (int, error) {
	return readNextNumber(s.root)
}

func readNextNumber(root string) (int, error) {
	path := filepath.Join(root, metaDir, nextNumFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run — scan projects for backward compatibility
		n, err := scanNextNumber(root)
		if err != nil {
			return 0, err
		}
		// Initialize the counter file so future calls are O(1)
		dir := filepath.Join(root, metaDir)
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			return 0, fmt.Errorf("create meta dir: %w", mkErr)
		}
		if wErr := writeAtomic(path, []byte(strconv.Itoa(n))); wErr != nil {
			// Non-fatal: we can still return the correct number
			_ = wErr
		}
		return n, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read next number: %w", err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Corrupted file — repair and fall back to scanning
		n, err = scanNextNumber(root)
		if err != nil {
			return 0, err
		}
		writeNextNumber(root, n) // best-effort repair
		return n, nil
	}
	return n, nil
}

func scanNextNumber(root string) (int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, fmt.Errorf("scan next number: %w", err)
	}
	maxN := 0
	for _, e := range entries {
		if !e.IsDir() || e.Name() == metaDir || e.Name() == ".trash" {
			continue
		}
		p, err := readProjectFile(root, e.Name())
		if err != nil {
			continue
		}
		if p.Number > maxN {
			maxN = p.Number
		}
	}
	return maxN + 1, nil
}

func writeNextNumber(root string, n int) error {
	dir := filepath.Join(root, metaDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create meta dir: %w", err)
	}
	path := filepath.Join(dir, nextNumFile)
	return writeAtomic(path, []byte(strconv.Itoa(n)))
}

func (s *FileStore) AdvanceNextNumber() error {
	n, err := readNextNumber(s.root)
	if err != nil {
		return err
	}
	return writeNextNumber(s.root, n+1)
}

func (s *FileStore) SaveProject(p types.Project) error {
	unlock, err := s.lockProject(p.ID)
	if err != nil {
		return err
	}
	defer unlock()

	dir := s.projectDir(p.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, "project.yaml"), p)
}

func (s *FileStore) SaveStep(st types.Step) error {
	unlock, err := s.lockProject(st.ProjectID)
	if err != nil {
		return err
	}
	defer unlock()

	return s.saveStep(st)
}

func (s *FileStore) saveStep(st types.Step) error {
	dir := s.stepsDir(st.ProjectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, st.ID+".yaml"), st)
}

func (s *FileStore) SaveBlocker(b types.Blocker) error {
	if b.StepID == "" {
		return fmt.Errorf("blocker has no StepID")
	}

	unlock, err := s.lockProject(b.ProjectID)
	if err != nil {
		return err
	}
	defer unlock()

	steps, err := s.GetSteps(b.ProjectID)
	if err != nil {
		return err
	}
	for i, st := range steps {
		if st.ID == b.StepID {
			found := false
			for j, existing := range st.Blockers {
				if existing.ID == b.ID {
					steps[i].Blockers[j] = b
					found = true
					break
				}
			}
			if !found {
				steps[i].Blockers = append(steps[i].Blockers, b)
			}
			if b.Status != types.BlockerResolved {
				steps[i].Status = types.StepBlocked
			}
			return s.saveStep(steps[i])
		}
	}
	return fmt.Errorf("step %q not found", b.StepID)
}

func (s *FileStore) SaveDecision(d types.Decision) error {
	unlock, err := s.lockProject(d.ProjectID)
	if err != nil {
		return err
	}
	defer unlock()

	dir := s.decisionsDir(d.ProjectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeYAMLAtomic(filepath.Join(dir, d.ID+".yaml"), d)
}

// ClosePlan previews CloseProject without mutating anything.
//
// Read-only by contract: it resolves and reads, never saves. That is the whole
// point — a caller must be able to show a human what an irreversible bulk close
// would do, so this cannot be the thing that changes the project.
func (s *FileStore) ClosePlan(ref string) (*types.ClosePlan, error) {
	pd, err := s.ResolveProject(ref)
	if err != nil {
		return nil, err
	}
	return buildClosePlan(*pd), nil
}

// buildClosePlan assembles the preview from already-loaded project data.
// Shared by FileStore and MockStore so both describe a close identically.
func buildClosePlan(pd types.ProjectData) *types.ClosePlan {
	plan := &types.ClosePlan{
		ProjectID:    pd.Project.ID,
		ProjectTitle: pd.Project.Title,
		Steps:        []types.ClosePlanStep{},
		Blockers:     []types.ClosePlanBlocker{},
	}
	for _, st := range pd.Steps {
		if st.Status == types.StepDone {
			continue // already closed, nothing to do
		}
		plan.Steps = append(plan.Steps, types.ClosePlanStep{
			ID:     st.ID,
			Title:  st.Title,
			Status: st.Status,
		})
		for _, b := range st.Blockers {
			if b.Status == types.BlockerResolved {
				continue
			}
			plan.Blockers = append(plan.Blockers, types.ClosePlanBlocker{
				ID:       b.ID,
				Title:    b.Title,
				Reason:   b.Reason,
				StepID:   st.ID,
				StepName: st.Title,
			})
		}
	}
	return plan
}

// CloseProject force-completes open steps, marks the project completed
// and records the reason. One call instead of N lifecycle transitions.
// Each op locks individually (no nested locks); the daemon serializes
// concurrent closes with its own mutex.
//
// confirm gates the irreversible half: false returns the plan and changes
// nothing. The gate lives here, in the store, not only in the MCP tool — every
// caller reaches this through an adapter, and a gate the adapter can skip is
// not a gate.
func (s *FileStore) CloseProject(ref, reason string, confirm bool) (*types.ClosePlan, error) {
	pd, err := s.ResolveProject(ref)
	if err != nil {
		return nil, err
	}
	if pd.Project.Status == types.StatusCompleted {
		return nil, fmt.Errorf("project #%d already completed", pd.Project.Number)
	}

	plan := buildClosePlan(*pd)
	if !confirm {
		return plan, nil
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("reason is required to confirm a close: it is recorded as the project decision")
	}

	now := types.NowTimestamp()
	closed := 0
	for _, st := range pd.Steps {
		if st.Status == types.StepDone {
			continue
		}
		st.Status = types.StepDone
		st.UpdatedAt = now
		if err := s.SaveStep(st); err != nil {
			return nil, fmt.Errorf("close step %q: %w", st.ID, err)
		}
		closed++
	}
	pd.Project.Status = types.StatusCompleted
	pd.Project.CompletedAt = now
	pd.Project.UpdatedAt = now
	if err := s.SaveProject(pd.Project); err != nil {
		return nil, fmt.Errorf("complete project: %w", err)
	}
	if err := s.SaveDecision(types.Decision{
		ID: "closed", Title: "Closed: " + reason, Reason: reason,
		Date: now, ProjectID: pd.Project.ID,
	}); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *FileStore) DeleteProject(id string) error {
	unlock, err := s.lockProject(id)
	if err != nil {
		return err
	}
	defer unlock()

	trashDir := filepath.Join(s.root, ".trash")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return fmt.Errorf("create trash: %w", err)
	}

	// Snapshot first: the rename below is the point of no return, and a
	// failed backup must stop the delete rather than let data go unrecorded.
	runDir, err := s.backupRunDir()
	if err != nil {
		return err
	}
	if err := s.backupProjectDir(id, runDir); err != nil {
		return fmt.Errorf("backup project before delete: %w", err)
	}
	defer s.pruneBackups()

	src := s.projectDir(id)
	dst := filepath.Join(trashDir, fmt.Sprintf("%s-%d", id, time.Now().Unix()))
	return os.Rename(src, dst)
}

func (s *FileStore) DeleteStep(projectID, stepID string) error {
	unlock, err := s.lockProject(projectID)
	if err != nil {
		return err
	}
	defer unlock()

	if err := s.backupBeforeDelete(projectID, filepath.Join("steps", stepID+".yaml")); err != nil {
		return err
	}
	defer s.pruneBackups()

	if err := os.Remove(filepath.Join(s.stepsDir(projectID), stepID+".yaml")); err != nil {
		return err
	}
	s.touchProject(projectID)
	return nil
}

func (s *FileStore) DeleteBlocker(projectID, stepID, blockerID string) error {
	unlock, err := s.lockProject(projectID)
	if err != nil {
		return err
	}
	defer unlock()

	steps, err := s.GetSteps(projectID)
	if err != nil {
		return err
	}
	for i, st := range steps {
		if st.ID == stepID {
			for j, b := range st.Blockers {
				if b.ID == blockerID {
					if err := s.backupBeforeDelete(projectID, filepath.Join("steps", stepID+".yaml")); err != nil {
						return err
					}
					defer s.pruneBackups()
					steps[i].Blockers = append(st.Blockers[:j], st.Blockers[j+1:]...)
					if !domain.HasUnresolvedBlockers(steps[i].Blockers) {
						steps[i].Status = types.StepTodo
					}
					if err := s.saveStep(steps[i]); err != nil {
						return err
					}
					s.touchProject(projectID)
					return nil
				}
			}
			return fmt.Errorf("blocker %q not found in step %q", blockerID, stepID)
		}
	}
	return fmt.Errorf("step %q not found", stepID)
}

// touchProject updates the project's UpdatedAt timestamp to now.
// Must be called while the project lock is held. Errors are logged but not returned.
func (s *FileStore) touchProject(projectID string) {
	projectPath := filepath.Join(s.projectDir(projectID), "project.yaml")
	data, err := os.ReadFile(projectPath)
	if err != nil {
		slog.Warn("touch project read", "project", projectID, "error", err)
		return
	}
	var p types.Project
	if err := yaml.Unmarshal(data, &p); err != nil {
		slog.Warn("touch project parse", "project", projectID, "error", err)
		return
	}
	p.UpdatedAt = types.NowTimestamp()
	if err := writeYAMLAtomic(projectPath, p); err != nil {
		slog.Warn("touch project write", "project", projectID, "error", err)
	}
}

func (s *FileStore) DeleteDecision(projectID, decisionID string) error {
	unlock, err := s.lockProject(projectID)
	if err != nil {
		return err
	}
	defer unlock()

	if err := s.backupBeforeDelete(projectID, filepath.Join("decisions", decisionID+".yaml")); err != nil {
		return err
	}
	defer s.pruneBackups()

	if err := os.Remove(filepath.Join(s.decisionsDir(projectID), decisionID+".yaml")); err != nil {
		return err
	}
	s.touchProject(projectID)
	return nil
}

// splitTrashName separates `<project-id>-<unix-timestamp>`. The id itself
// contains dashes (UUID), so the timestamp is the part after the LAST dash.
func splitTrashName(trashName string) (projectID string, deletedAt time.Time, ok bool) {
	idx := strings.LastIndex(trashName, "-")
	if idx < 1 {
		return "", time.Time{}, false
	}
	ts, err := strconv.ParseInt(trashName[idx+1:], 10, 64)
	if err != nil {
		return "", time.Time{}, false
	}
	return trashName[:idx], time.Unix(ts, 0).UTC(), true
}

func (s *FileStore) TrashList() ([]types.TrashItem, error) {
	trashDir := filepath.Join(s.root, ".trash")
	entries, err := os.ReadDir(trashDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	items := make([]types.TrashItem, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		projectID, deletedAt, ok := splitTrashName(name)
		if !ok {
			continue
		}
		item := types.TrashItem{
			TrashName: name,
			ProjectID: projectID,
			DeletedAt: types.NewTimestamp(deletedAt),
		}
		// Best effort: a trashed project.yaml carries number and title.
		// An unreadable one stays listable and restorable by trash name.
		// Note: the trash entry dir holds project.yaml directly, so the
		// trash dir is the root and the entry name is the id here.
		if p, err := readProjectFile(trashDir, name); err == nil {
			item.Number = p.Number
			item.Title = p.Title
		} else {
			slog.Warn("trash item unreadable", "trash", name, "error", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *FileStore) TrashRestore(ref string) error {
	trashDir := filepath.Join(s.root, ".trash")
	items, err := s.TrashList()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("trash item %q not found: trash is empty (run trash list)", ref)
	}

	// Exact trash name always wins: it is unambiguous by construction.
	for _, it := range items {
		if it.TrashName == ref {
			return s.restoreTrashItem(trashDir, it)
		}
	}

	var candidates []types.TrashItem
	if n, err := strconv.Atoi(strings.TrimSpace(ref)); err == nil {
		for _, it := range items {
			if it.Number == n {
				candidates = append(candidates, it)
			}
		}
	} else {
		lower := strings.ToLower(strings.TrimSpace(ref))
		for _, it := range items {
			if strings.Contains(strings.ToLower(it.Title), lower) {
				candidates = append(candidates, it)
			}
		}
	}
	switch len(candidates) {
	case 0:
		return fmt.Errorf("trash item %q not found: run trash list to see trash names", ref)
	case 1:
		return s.restoreTrashItem(trashDir, candidates[0])
	default:
		names := make([]string, 0, len(candidates))
		for _, c := range candidates {
			names = append(names, fmt.Sprintf("%s (#%d %q)", c.TrashName, c.Number, c.Title))
		}
		return fmt.Errorf("trash restore %q is ambiguous, %d candidates and nothing restored: %s",
			ref, len(candidates), strings.Join(names, "; "))
	}
}

// restoreTrashItem moves one trash dir back into the live tree. It refuses
// rather than merges: if a live project already holds the number, or the
// target dir exists, restoring would silently corrupt the live set.
func (s *FileStore) restoreTrashItem(trashDir string, it types.TrashItem) error {
	if it.Number != 0 {
		live, err := s.ListProjects()
		if err == nil {
			for _, p := range live {
				if p.Number == it.Number {
					return fmt.Errorf("cannot restore %q: live project #%d %q already holds that number",
						it.TrashName, p.Number, p.Title)
				}
			}
		}
	}
	src := filepath.Join(trashDir, it.TrashName)
	if info, err := os.Stat(src); err != nil {
		return fmt.Errorf("trash item %q not found: %w", it.TrashName, err)
	} else if !info.IsDir() {
		return fmt.Errorf("trash item %q is not a directory", it.TrashName)
	}
	dst := s.projectDir(it.ProjectID)
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("cannot restore %q: live project dir %q already exists", it.TrashName, it.ProjectID)
	}
	return os.Rename(src, dst)
}

func (s *FileStore) TrashClean() error {
	trashDir := filepath.Join(s.root, ".trash")
	return os.RemoveAll(trashDir)
}

// Check walks the store root and reports integrity. Read-only: it never
// writes, so a check cannot be the thing that corrupts the store.
//
// This used to live in the CLI, which walked whatever PM_DIR pointed at
// while the daemon served something else. Now the single writer walks its
// own root and every caller reads one verdict.
func (s *FileStore) Check() (*types.DoctorReport, error) {
	root := s.root
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("store not found: %s (run pm init)", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read store root: %w", err)
	}

	rep := &types.DoctorReport{Root: root}
	known := map[string]bool{".trash": true, "_meta": true}
	countStamps := func(rep *types.DoctorReport, stamps ...types.Timestamp) {
		for _, ts := range stamps {
			switch _, invalid := ts.Invalid(); {
			case invalid:
				rep.BrokenTimestamps++
			case ts.IsLegacy():
				rep.LegacyTimestamps++
			}
		}
	}

	for _, e := range entries {
		if !e.IsDir() || known[e.Name()] {
			continue
		}
		projDir := filepath.Join(root, e.Name())
		projectFile := filepath.Join(projDir, "project.yaml")
		if _, err := os.Stat(projectFile); os.IsNotExist(err) {
			rep.Orphans = append(rep.Orphans, e.Name())
			continue
		}
		data, err := os.ReadFile(projectFile)
		if err != nil {
			rep.Issues = append(rep.Issues, fmt.Sprintf("%s: read error: %v", e.Name(), err))
			continue
		}
		var p types.Project
		if err := yaml.Unmarshal(data, &p); err != nil {
			rep.Issues = append(rep.Issues, fmt.Sprintf("%s: YAML parse error: %v", e.Name(), err))
			continue
		}
		line := types.DoctorProjectLine{Number: p.Number, Title: p.Title, ID: p.ID}
		countStamps(rep, p.CreatedAt, p.UpdatedAt, p.CompletedAt)

		if stepEntries, err := os.ReadDir(filepath.Join(projDir, "steps")); err == nil {
			for _, se := range stepEntries {
				if se.IsDir() || filepath.Ext(se.Name()) != ".yaml" {
					continue
				}
				stepData, err := os.ReadFile(filepath.Join(projDir, "steps", se.Name()))
				if err != nil {
					rep.Issues = append(rep.Issues, fmt.Sprintf("%s step %s: read error: %v", e.Name(), se.Name(), err))
					continue
				}
				var step types.Step
				if err := yaml.Unmarshal(stepData, &step); err != nil {
					rep.Issues = append(rep.Issues, fmt.Sprintf("%s step %s: YAML parse error: %v", e.Name(), se.Name(), err))
					continue
				}
				countStamps(rep, step.CreatedAt, step.UpdatedAt)
				for _, bl := range step.Blockers {
					countStamps(rep, bl.CreatedAt, bl.UpdatedAt)
				}
				if step.ProjectID != p.ID {
					rep.Issues = append(rep.Issues, fmt.Sprintf("%s step %s: project_id mismatch (%s != %s)", e.Name(), se.Name(), step.ProjectID, p.ID))
				}
				line.Steps++
				line.Blockers += len(step.Blockers)
			}
		}
		rep.TotalSteps += line.Steps
		rep.TotalBlockers += line.Blockers

		if decEntries, err := os.ReadDir(filepath.Join(projDir, "decisions")); err == nil {
			for _, de := range decEntries {
				if de.IsDir() || filepath.Ext(de.Name()) != ".yaml" {
					continue
				}
				decData, err := os.ReadFile(filepath.Join(projDir, "decisions", de.Name()))
				if err != nil {
					rep.Issues = append(rep.Issues, fmt.Sprintf("%s decision %s: read error: %v", e.Name(), de.Name(), err))
					continue
				}
				var dec types.Decision
				if err := yaml.Unmarshal(decData, &dec); err != nil {
					rep.Issues = append(rep.Issues, fmt.Sprintf("%s decision %s: YAML parse error: %v", e.Name(), de.Name(), err))
					continue
				}
				countStamps(rep, dec.Date)
				if dec.ProjectID != p.ID {
					rep.Issues = append(rep.Issues, fmt.Sprintf("%s decision %s: project_id mismatch (%s != %s)", e.Name(), de.Name(), dec.ProjectID, p.ID))
				}
				line.Decisions++
			}
		}
		rep.TotalDecisions += line.Decisions
		rep.Projects = append(rep.Projects, line)
	}
	return rep, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (s *FileStore) projectDir(id string) string {
	return filepath.Join(s.root, id)
}

func (s *FileStore) stepsDir(id string) string {
	return filepath.Join(s.root, id, "steps")
}

func (s *FileStore) decisionsDir(id string) string {
	return filepath.Join(s.root, id, "decisions")
}

// loadProjectData attaches steps and decisions to a project metadata struct
// without re-reading project.yaml (the caller already parsed it).
func (s *FileStore) loadProjectData(p types.Project) (*types.ProjectData, error) {
	steps, err := s.GetSteps(p.ID)
	if err != nil {
		slog.Warn("load steps for project", "project", p.ID, "error", err)
		steps = nil
	}
	decisions, err := s.GetDecisions(p.ID)
	if err != nil {
		slog.Warn("load decisions for project", "project", p.ID, "error", err)
		decisions = nil
	}
	return &types.ProjectData{
		Project:   p,
		Steps:     steps,
		Decisions: decisions,
	}, nil
}

// readProjectFile reads a single project.yaml from disk, given the store root and project ID.
func readProjectFile(root, id string) (*types.Project, error) {
	path := filepath.Join(root, id, "project.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read project %s: %w", id, err)
	}
	var p types.Project
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse project %s: %w", id, err)
	}
	return &p, nil
}

func (s *FileStore) readProject(id string) (*types.Project, error) {
	return readProjectFile(s.root, id)
}

func (s *FileStore) GetSteps(projectID string) ([]types.Step, error) {
	dir := s.stepsDir(projectID)
	return readYAMLDir[types.Step](dir)
}

func (s *FileStore) GetBlockers(projectID string) ([]types.Blocker, error) {
	steps, err := s.GetSteps(projectID)
	if err != nil {
		return nil, err
	}
	var blockers []types.Blocker
	for _, st := range steps {
		blockers = append(blockers, st.Blockers...)
	}
	return blockers, nil
}

func (s *FileStore) GetDecisions(projectID string) ([]types.Decision, error) {
	dir := s.decisionsDir(projectID)
	return readYAMLDir[types.Decision](dir)
}

// ---------------------------------------------------------------------------
// Locking
// ---------------------------------------------------------------------------

// lockProject acquires an exclusive POSIX file lock (flock) on the project
// directory. It blocks until the lock is acquired or an error occurs.
// The lock is automatically released when the process exits.
// Returns an unlock function that MUST be called (typically with defer).
func (s *FileStore) lockProject(projectID string) (unlock func(), err error) {
	lockPath := filepath.Join(s.projectDir(projectID), ".pm.lock")

	// Ensure the project directory exists for the lock file.
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", lockPath, err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("flock %s: %w", lockPath, err)
	}

	var once bool
	return func() {
		if once {
			return
		}
		once = true
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// ---------------------------------------------------------------------------
// Atomic YAML write
// ---------------------------------------------------------------------------

// writeYAMLAtomic marshals v to YAML and writes it to path atomically.
// It writes to a temporary file in the same directory (same filesystem),
// calls fsync on both the file and its parent directory, then renames into
// place. If the process crashes mid-write, the target file remains intact.
func writeYAMLAtomic(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return writeAtomic(path, data)
}

// writeAtomic writes data to path atomically with full fsync.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)

	// Temp file in the same directory (same filesystem → rename is atomic).
	tmp, err := os.CreateTemp(dir, ".tmp-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp on failure.
	cleanup := true
	defer func() {
		if cleanup {
			tmp.Close()
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}

	// fsync data to disk.
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	// fsync parent directory so the new directory entry is durable.
	cleanup = false
	return fsyncDir(dir)
}

// fsyncDir opens dir and calls Sync() on it, ensuring the directory entry
// for a newly renamed file is persisted to disk.
func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = d.Sync()
	d.Close()
	return err
}

// ---------------------------------------------------------------------------
// Generic YAML directory reader
// ---------------------------------------------------------------------------

func readYAMLDir[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // empty directory = no data
		}
		return nil, err
	}
	var items []T
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		fp := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(fp)
		if err != nil {
			slog.Warn("cannot read YAML file", "path", fp, "error", err)
			continue
		}
		var item T
		if err := yaml.Unmarshal(data, &item); err != nil {
			slog.Warn("cannot parse YAML file", "path", fp, "error", err)
			continue
		}
		items = append(items, item)
	}
	return items, nil
}
