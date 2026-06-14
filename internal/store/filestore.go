package store

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
	"github.com/xvantz/pm/internal/types"
)

// FileStore reads/writes project data from YAML files on disk.
type FileStore struct {
	root string // e.g. ./pm/projects
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

func (s *FileStore) projectDir(id string) string {
	return filepath.Join(s.root, id)
}

func (s *FileStore) stepsDir(id string) string {
	return filepath.Join(s.root, id, "steps")
}

func (s *FileStore) decisionsDir(id string) string {
	return filepath.Join(s.root, id, "decisions")
}

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
	steps, err := s.GetSteps(id)
	if err != nil {
		steps = nil
	}
	decisions, err := s.GetDecisions(id)
	if err != nil {
		decisions = nil
	}
	return &types.ProjectData{
		Project:   *project,
		Steps:     steps,
		Decisions: decisions,
	}, nil
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
				// Already have project metadata from ListProjects — avoid re-read
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
		}
		return nil, fmt.Errorf("project #%d not found", n)
	}

	// Try as exact UUID first (fast path)
	if pd, err := s.GetProject(ref); err == nil {
		return pd, nil
	}

	// Try as UUID prefix (scan required)
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	var matches []types.Project
	for _, p := range projects {
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
	return s.GetProject(matches[0].ID)
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

func (s *FileStore) SaveProject(p types.Project) error {
	dir := s.projectDir(p.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeYAML(filepath.Join(dir, "project.yaml"), p)
}

func (s *FileStore) SaveStep(st types.Step) error {
	dir := s.stepsDir(st.ProjectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeYAML(filepath.Join(dir, st.ID+".yaml"), st)
}

func (s *FileStore) SaveBlocker(b types.Blocker) error {
	if b.StepID == "" {
		return fmt.Errorf("blocker has no StepID")
	}
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
			// Adding/updating a blocker marks the step as blocked (unless resolved)
			if b.Status != types.BlockerResolved {
				steps[i].Status = types.StepBlocked
			}
			return s.SaveStep(steps[i])
		}
	}
	return fmt.Errorf("step %q not found", b.StepID)
}

func (s *FileStore) SaveDecision(d types.Decision) error {
	dir := s.decisionsDir(d.ProjectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeYAML(filepath.Join(dir, d.ID+".yaml"), d)
}

func (s *FileStore) DeleteProject(id string) error {
	return os.RemoveAll(s.projectDir(id))
}

func (s *FileStore) DeleteStep(projectID, stepID string) error {
	return os.Remove(filepath.Join(s.stepsDir(projectID), stepID+".yaml"))
}

func (s *FileStore) DeleteBlocker(projectID, stepID, blockerID string) error {
	steps, err := s.GetSteps(projectID)
	if err != nil {
		return err
	}
	for i, st := range steps {
		if st.ID == stepID {
			for j, b := range st.Blockers {
				if b.ID == blockerID {
					steps[i].Blockers = append(st.Blockers[:j], st.Blockers[j+1:]...)
					// If no more blockers, step goes back to todo
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
					return s.SaveStep(steps[i])
				}
			}
			return fmt.Errorf("blocker %q not found in step %q", blockerID, stepID)
		}
	}
	return fmt.Errorf("step %q not found", stepID)
}

func (s *FileStore) DeleteDecision(projectID, decisionID string) error {
	return os.Remove(filepath.Join(s.decisionsDir(projectID), decisionID+".yaml"))
}

func readYAMLDir[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
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

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
