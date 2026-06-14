package store

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"gopkg.in/yaml.v3"

	"github.com/xvantz/pm/internal/types"
)

// FileStore reads/writes project data from YAML files on disk.
// All write operations use POSIX file locks (flock) on the project directory
// to coordinate between concurrent CLI and MCP processes. Writes are atomic:
// data is written to a temp file, synced to disk, then renamed into place.
type FileStore struct {
	root string // e.g. ./pm/projects
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
	projects, err := s.ListProjects()
	if err != nil {
		return 0, fmt.Errorf("list projects for next number: %w", err)
	}
	maxN := 0
	for _, p := range projects {
		if p.Number > maxN {
			maxN = p.Number
		}
	}
	return maxN + 1, nil
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

func (s *FileStore) DeleteProject(id string) error {
	unlock, err := s.lockProject(id)
	if err != nil {
		return err
	}
	defer unlock()

	return os.RemoveAll(s.projectDir(id))
}

func (s *FileStore) DeleteStep(projectID, stepID string) error {
	unlock, err := s.lockProject(projectID)
	if err != nil {
		return err
	}
	defer unlock()

	return os.Remove(filepath.Join(s.stepsDir(projectID), stepID+".yaml"))
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
					steps[i].Blockers = append(st.Blockers[:j], st.Blockers[j+1:]...)
					stillBlocked := false
					for _, remaining := range steps[i].Blockers {
						if remaining.Status == types.BlockerWaiting || remaining.Status == types.BlockerActive {
							stillBlocked = true
							break
						}
					}
					if !stillBlocked {
						steps[i].Status = types.StepTodo
					}
					return s.saveStep(steps[i])
				}
			}
			return fmt.Errorf("blocker %q not found in step %q", blockerID, stepID)
		}
	}
	return fmt.Errorf("step %q not found", stepID)
}

func (s *FileStore) DeleteDecision(projectID, decisionID string) error {
	unlock, err := s.lockProject(projectID)
	if err != nil {
		return err
	}
	defer unlock()

	return os.Remove(filepath.Join(s.decisionsDir(projectID), decisionID+".yaml"))
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
		steps = nil
	}
	decisions, err := s.GetDecisions(p.ID)
	if err != nil {
		decisions = nil
	}
	return &types.ProjectData{
		Project:   p,
		Steps:     steps,
		Decisions: decisions,
	}, nil
}

func (s *FileStore) readProject(id string) (*types.Project, error) {
	path := filepath.Join(s.projectDir(id), "project.yaml")
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
