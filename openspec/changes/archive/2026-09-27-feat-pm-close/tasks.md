# Tasks: feat-pm-close

## Phase 1: Spec (this fix)

- [x] 1.1 Delta `mcp-server`: ADDED `close_project` + `delete_project` tools
  **DoD:** `openspec validate feat-pm-close --type change` green
- [x] 1.2 Delta `project-store`: ADDED `CloseProject` store contract
  **DoD:** same validate green
- [x] 1.3 tasks.md with DoD per task (was missing entirely)
  **DoD:** `openspec list` shows task count, not "No tasks"

## Phase 2: Code (already on this branch)

- [x] 2.1 `CloseProject` in FileStore + MockStore + Store interface
- [x] 2.2 API `POST /api/projects/{ref}/close`, CLI `pm project close`, client + apistore
- [x] 2.3 MCP `close_project` + `delete_project` with `type: object` schemas
- [x] 2.4 Tests: server, store, MCP registration
  **DoD:** `CGO_ENABLED=0 go test ./...` green (verified 2026-09-27 on sibling branch)

## Phase 3: Merge + archive

- [ ] 3.1 PR to main, CI green
  **DoD:** Forgejo run success on the PR
- [ ] 3.2 Archive, validate clean
  **DoD:** `openspec list` without feat-pm-close; `validate --all --strict` zero-fail
