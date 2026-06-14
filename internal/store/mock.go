package store

import (
	"fmt"
	"strconv"

	"github.com/xvantz/pm/internal/types"
)

// MockStore provides realistic mock data for development/testing.
type MockStore struct {
	projects map[string]*types.ProjectData
}

func NewMockStore() *MockStore {
	s := &MockStore{projects: make(map[string]*types.ProjectData)}
	s.seed()
	return s
}

func (s *MockStore) ListProjects() ([]types.Project, error) {
	var list []types.Project
	for _, pd := range s.projects {
		list = append(list, pd.Project)
	}
	return list, nil
}

func (s *MockStore) GetProject(id string) (*types.ProjectData, error) {
	pd, ok := s.projects[id]
	if !ok {
		return nil, fmt.Errorf("project %q not found", id)
	}
	return pd, nil
}

func (s *MockStore) ResolveProject(ref string) (*types.ProjectData, error) {
	if n, err := strconv.Atoi(ref); err == nil {
		for _, pd := range s.projects {
			if pd.Project.Number == n {
				return pd, nil
			}
		}
		return nil, fmt.Errorf("project #%d not found", n)
	}

	// Try as exact UUID first
	if pd, ok := s.projects[ref]; ok {
		return pd, nil
	}

	// Try as UUID prefix
	var matches []*types.ProjectData
	for _, pd := range s.projects {
		if len(pd.Project.ID) >= len(ref) && pd.Project.ID[:len(ref)] == ref {
			matches = append(matches, pd)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("project %q not found", ref)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous project prefix %q matches %d projects", ref, len(matches))
	}
	return matches[0], nil
}

func (s *MockStore) NextNumber() (int, error) {
	maxN := 0
	for _, pd := range s.projects {
		if pd.Project.Number > maxN {
			maxN = pd.Project.Number
		}
	}
	return maxN + 1, nil
}

func (s *MockStore) GetSteps(projectID string) ([]types.Step, error) {
	pd, ok := s.projects[projectID]
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectID)
	}
	return pd.Steps, nil
}

func (s *MockStore) GetBlockers(projectID string) ([]types.Blocker, error) {
	pd, ok := s.projects[projectID]
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectID)
	}
	var blockers []types.Blocker
	for _, st := range pd.Steps {
		blockers = append(blockers, st.Blockers...)
	}
	return blockers, nil
}

func (s *MockStore) GetDecisions(projectID string) ([]types.Decision, error) {
	pd, ok := s.projects[projectID]
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectID)
	}
	return pd.Decisions, nil
}

func (s *MockStore) SaveProject(p types.Project) error {
	existing, ok := s.projects[p.ID]
	if ok {
		existing.Project = p
	} else {
		s.projects[p.ID] = &types.ProjectData{
			Project:   p,
			Steps:     []types.Step{},
			Decisions: []types.Decision{},
		}
	}
	return nil
}

func (s *MockStore) SaveStep(st types.Step) error {
	pd, ok := s.projects[st.ProjectID]
	if !ok {
		return fmt.Errorf("project %q not found", st.ProjectID)
	}
	for i, step := range pd.Steps {
		if step.ID == st.ID {
			pd.Steps[i] = st
			return nil
		}
	}
	pd.Steps = append(pd.Steps, st)
	return nil
}

func (s *MockStore) SaveBlocker(b types.Blocker) error {
	if b.StepID == "" {
		return fmt.Errorf("blocker has no StepID")
	}
	for _, pd := range s.projects {
		if pd.Project.ID != b.ProjectID {
			continue
		}
		for i, st := range pd.Steps {
			if st.ID == b.StepID {
				found := false
				for j, existing := range st.Blockers {
					if existing.ID == b.ID {
						pd.Steps[i].Blockers[j] = b
						found = true
						break
					}
				}
				if !found {
					pd.Steps[i].Blockers = append(pd.Steps[i].Blockers, b)
				}
				// Adding/updating a blocker marks the step as blocked (unless resolved)
				if b.Status != types.BlockerResolved {
					pd.Steps[i].Status = types.StepBlocked
				}
				return nil
			}
		}
		return fmt.Errorf("step %q not found in project %q", b.StepID, b.ProjectID)
	}
	return fmt.Errorf("project %q not found", b.ProjectID)
}

func (s *MockStore) SaveDecision(d types.Decision) error {
	for _, pd := range s.projects {
		if pd.Project.ID != d.ProjectID {
			continue
		}
		found := false
		for i, existing := range pd.Decisions {
			if existing.ID == d.ID {
				pd.Decisions[i] = d
				found = true
				break
			}
		}
		if !found {
			pd.Decisions = append(pd.Decisions, d)
		}
		return nil
	}
	return fmt.Errorf("project %q not found", d.ProjectID)
}

func (s *MockStore) DeleteProject(id string) error {
	delete(s.projects, id)
	return nil
}

func (s *MockStore) DeleteStep(projectID, stepID string) error {
	pd, ok := s.projects[projectID]
	if !ok {
		return nil
	}
	for i, st := range pd.Steps {
		if st.ID == stepID {
			pd.Steps = append(pd.Steps[:i], pd.Steps[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *MockStore) DeleteBlocker(projectID, stepID, blockerID string) error {
	pd, ok := s.projects[projectID]
	if !ok {
		return nil
	}
	for i, st := range pd.Steps {
		if st.ID == stepID {
			for j, b := range st.Blockers {
				if b.ID == blockerID {
					pd.Steps[i].Blockers = append(st.Blockers[:j], st.Blockers[j+1:]...)
					// If no more blockers, step goes back to todo
					stillBlocked := false
					for _, remaining := range pd.Steps[i].Blockers {
						if remaining.Status == types.BlockerWaiting || remaining.Status == types.BlockerActive {
							stillBlocked = true
							break
						}
					}
					if !stillBlocked {
						pd.Steps[i].Status = types.StepTodo
					}
					return nil
				}
			}
			return nil
		}
	}
	return nil
}

func (s *MockStore) DeleteDecision(projectID, decisionID string) error {
	pd, ok := s.projects[projectID]
	if !ok {
		return nil
	}
	for i, d := range pd.Decisions {
		if d.ID == decisionID {
			pd.Decisions = append(pd.Decisions[:i], pd.Decisions[i+1:]...)
			return nil
		}
	}
	return nil
}

// GetRaw returns the internal project map for testing.
func (s *MockStore) GetRaw() map[string]*types.ProjectData {
	return s.projects
}

func (s *MockStore) seed() {
	today := "2026-06-14"
	yesterday := "2026-06-13"

	// AGH (#1)
	s.projects["0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a"] = &types.ProjectData{
		Project: types.Project{
			ID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a", Number: 1,
			Title: "AdGuard Home",
			Goal:  "Развернуть домашний DNS сервер с фильтрацией рекламы",
			Status: types.StatusActive,
			Tags:   []string{"infrastructure", "homelab", "networking"},
			CreatedAt: "2026-06-10", UpdatedAt: today,
		},
		Steps: []types.Step{
			{ID: "setup-caddy", Title: "Настроить Caddy reverse proxy", Status: types.StepDone, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a", CreatedAt: yesterday, Artifacts: []string{"docs/caddy-setup.md"}},
			{ID: "install-agh", Title: "Установить и настроить AGH", Status: types.StepDone, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a", CreatedAt: yesterday},
			{ID: "configure-dns", Title: "Настроить DNS маршрутизацию", Status: types.StepBlocked, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a", CreatedAt: today,
				Blockers: []types.Blocker{
					{ID: "router", Title: "Купить GL.iNet роутер", Reason: "Нет свободного бюджета", Status: types.BlockerWaiting, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a", StepID: "configure-dns", CreatedAt: "2026-06-10"},
				},
			},
			{ID: "test-dns", Title: "Протестировать фильтрацию на всех устройствах", Status: types.StepTodo, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a"},
			{ID: "vpn-access", Title: "Настроить DNS через Tailscale", Status: types.StepTodo, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a"},
		},
		Decisions: []types.Decision{
			{ID: "migrate-docker", Title: "Отказ от Docker в пользу NixOS Service", Reason: "NixOS Service проще сопровождать", Date: yesterday, ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a"},
			{ID: "use-agh", Title: "AdGuard Home вместо Pi-hole", Reason: "Лучшая поддержка DNS-over-HTTPS, более современный UI", Date: "2026-06-11", ProjectID: "0196f1a2-b3c4-7d5e-8f6a-9b0c1d2e3f4a"},
		},
	}

	// PM (#2)
	s.projects["0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"] = &types.ProjectData{
		Project: types.Project{
			ID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b", Number: 2,
			Title: "Project Memory (PM)",
			Goal:  "Создать систему долговременной памяти проектов",
			Status: types.StatusActive,
			Tags:   []string{"tooling", "infrastructure"},
			CreatedAt: "2026-06-13", UpdatedAt: today,
		},
		Steps: []types.Step{
			{ID: "write-spec", Title: "Написать спецификацию PM", Status: types.StepDone, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b", CreatedAt: yesterday},
			{ID: "review-spec", Title: "Согласовать спецификацию", Status: types.StepInProgress, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b", CreatedAt: today},
			{ID: "impl-briefing", Title: "Реализовать движок pm briefing", Status: types.StepTodo, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"},
			{ID: "impl-init", Title: "Реализовать pm init", Status: types.StepTodo, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"},
			{ID: "impl-crud", Title: "Реализовать project/step/blocker crud", Status: types.StepTodo, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"},
			{ID: "mcp-layer", Title: "MCP-обёртка над CLI", Status: types.StepTodo, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"},
			{ID: "migrate-projects", Title: "Перенести проекты из Obsidian в PM", Status: types.StepTodo, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"},
		},
		Decisions: []types.Decision{
			{ID: "cli-first", Title: "CLI на старте, MCP через пул-реквест", Reason: "Сначала проверить, что CLI решает проблему", Date: yesterday, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"},
			{ID: "go-lang", Title: "Go как язык реализации", Reason: "Один бинарник, без зависимостей", Date: yesterday, ProjectID: "0196f1a3-c4d5-7e6f-8a9b-0c1d2e3f4a5b"},
		},
	}

	// Navidrome (#3)
	s.projects["0196f1a4-d5e6-7f8a-9b0c-1d2e3f4a5b6c"] = &types.ProjectData{
		Project: types.Project{
			ID: "0196f1a4-d5e6-7f8a-9b0c-1d2e3f4a5b6c", Number: 3,
			Title: "Navidrome Music Collector",
			Goal:  "Python-сервис для авто-пополнения музыки с обогащёнными метаданными",
			Status: types.StatusIdea,
			Tags:   []string{"infrastructure", "media"},
			CreatedAt: "2026-06-12", UpdatedAt: yesterday,
		},
		Steps: []types.Step{
			{ID: "api-research", Title: "Разведка API источников (Spotify, Last.fm, Genius)", Status: types.StepTodo, ProjectID: "0196f1a4-d5e6-7f8a-9b0c-1d2e3f4a5b6c"},
			{ID: "prototype-fetcher", Title: "Прототип fetcher для одного источника", Status: types.StepTodo, ProjectID: "0196f1a4-d5e6-7f8a-9b0c-1d2e3f4a5b6c"},
		},
	}

	// DNS Infrastructure (#4)
	s.projects["0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d"] = &types.ProjectData{
		Project: types.Project{
			ID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d", Number: 4,
			Title: "DNS Инфраструктура",
			Goal:  "Избавить все устройства от рекламы через свой DNS-сервер",
			Status: types.StatusActive,
			Tags:   []string{"infrastructure", "homelab", "networking", "dns"},
			CreatedAt: "2026-06-08", UpdatedAt: today,
		},
		Steps: []types.Step{
			{ID: "install-agora", Title: "Развернуть AGH на NixOS", Status: types.StepDone, ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d", CreatedAt: "2026-06-10"},
			{ID: "configure-filters", Title: "Настроить DNS фильтры", Status: types.StepDone, ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d", CreatedAt: "2026-06-11"},
			{ID: "tailscale-access", Title: "Настроить доступ через Tailscale", Status: types.StepDone, ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d", CreatedAt: "2026-06-12"},
			{ID: "resolve-conflict", Title: "Решить конфликт :53 с libvirt", Status: types.StepDone, ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d", CreatedAt: "2026-06-13"},
			{ID: "buy-router", Title: "Купить GL.iNet роутер", Status: types.StepTodo, ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d"},
			{ID: "setup-wifi", Title: "Настроить свою WiFi-сеть", Status: types.StepBlocked, ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d",
				Blockers: []types.Blocker{
					{ID: "router", Title: "Купить GL.iNet роутер", Reason: "Нет свободного бюджета", Status: types.BlockerWaiting, ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d", StepID: "setup-wifi", CreatedAt: "2026-06-10"},
				},
			},
		},
		Decisions: []types.Decision{
			{ID: "agh-over-pihole", Title: "AdGuard Home", Reason: "DNS-over-HTTPS, современный UI", Date: "2026-06-09", ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d"},
			{ID: "bind-address", Title: "specify dns.bind_hosts для libvirt", Reason: "Конфликт :53 с libvirt resolved", Date: "2026-06-13", ProjectID: "0196f1a5-e6f7-7a8b-9c0d-1e2f3a4b5c6d"},
		},
	}

	// Forgejo (#5)
	s.projects["0196f1a6-f7a8-7b9c-0d1e-2f3a4b5c6d7e"] = &types.ProjectData{
		Project: types.Project{
			ID: "0196f1a6-f7a8-7b9c-0d1e-2f3a4b5c6d7e", Number: 5,
			Title: "Автономная кузница кода Forgejo + Nix",
			Goal:  "Полностью автономная git-инфраструктура на NixOS с CI/CD",
			Status: types.StatusActive,
			Tags:   []string{"infrastructure", "devops", "selfhosted"},
			CreatedAt: "2026-06-10", UpdatedAt: yesterday,
		},
		Steps: []types.Step{
			{ID: "deploy-forgejo", Title: "Развернуть Forgejo на NixOS", Status: types.StepDone, ProjectID: "0196f1a6-f7a8-7b9c-0d1e-2f3a4b5c6d7e", CreatedAt: "2026-06-10"},
			{ID: "setup-act-runner", Title: "Настроить act-runner для CI", Status: types.StepDone, ProjectID: "0196f1a6-f7a8-7b9c-0d1e-2f3a4b5c6d7e", CreatedAt: "2026-06-11"},
			{ID: "sync-github", Title: "Настроить синхронизацию с GitHub", Status: types.StepInProgress, ProjectID: "0196f1a6-f7a8-7b9c-0d1e-2f3a4b5c6d7e", CreatedAt: yesterday},
			{ID: "backup-strategy", Title: "Настроить автоматический backup", Status: types.StepTodo, ProjectID: "0196f1a6-f7a8-7b9c-0d1e-2f3a4b5c6d7e"},
		},
	}

	// KeePassXC (#6)
	s.projects["0196f1a7-8a9b-7c0d-1e2f-3a4b5c6d7e8f"] = &types.ProjectData{
		Project: types.Project{
			ID: "0196f1a7-8a9b-7c0d-1e2f-3a4b5c6d7e8f", Number: 6,
			Title: "KeePassXC Password Manager",
			Goal:  "Настроить менеджер паролей с passkey support",
			Status: types.StatusPaused,
			Tags:   []string{"security", "tooling"},
			CreatedAt: "2026-06-11", UpdatedAt: "2026-06-13",
		},
		Steps: []types.Step{
			{ID: "evaluate-pm", Title: "Сравнить pass vs KeePassXC", Status: types.StepDone, ProjectID: "0196f1a7-8a9b-7c0d-1e2f-3a4b5c6d7e8f", CreatedAt: "2026-06-11"},
			{ID: "setup-keepass", Title: "Установить и настроить KeePassXC", Status: types.StepTodo, ProjectID: "0196f1a7-8a9b-7c0d-1e2f-3a4b5c6d7e8f"},
		},
		Decisions: []types.Decision{
			{ID: "choose-keepass", Title: "KeePassXC вместо pass", Reason: "pass не поддерживает passkey", Date: "2026-06-11", ProjectID: "0196f1a7-8a9b-7c0d-1e2f-3a4b5c6d7e8f"},
		},
	}
}
