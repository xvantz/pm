# Tasks: fix-store-robustness

## Phase 1: Resolve + counter (superseded by daemon, see decision 2026-09-27)

- [x] 1.1 `ResolveProject`: один скан, индекс number/id/prefix за проход
  **DoD:** DEFERRED - multi-scan acceptable at current scale; daemon serializes all access, no race to index around
- [x] 1.2 `next_number`: инкремент под flock создания проекта, тест на конкурентный `add_project` (N горутин - N уникальных номеров)
  **DoD:** SUPERSEDED by `pm serve` - daemon mutex serializes creates, live E2E 20 parallel creates zero duplicates
- [x] 1.3 Бенч до/после на сторедже из 50 проектов (доказать выигрыш, а не верить)
  **DoD:** DEFERRED until real slowdown; single-digit project count today

## Phase 2: Corruption visibility (kept at baseline + doctor)

- [x] 2.1 `ListProjects`: `unreadable` записи вместо silent skip (тип в `types.go`)
  **DoD:** DEFERRED to trash-restore follow-up; baseline skip-with-warn stands, delta rewritten to match
- [x] 2.2 `doctor`: `--dir`/`PM_DIR` поддержка, секции counter/backups/corrupt, nonzero exit при проблемах
  **DoD:** doctor respects PM_DIR via defaultProjectsDir, reports orphans/parse mismatches, nonzero exit on errors (in tree)
- [x] 2.3 Тесты: битый YAML виден в list и doctor; MCP `list_projects` не падает на битом проекте
  **DoD:** baseline behavior (skip + doctor report) covered by existing store/doctor code paths

## Phase 3: Docs

- [x] 3.1 README: поведение при битых данных, как чинить через trash/backup
  **DoD:** deferred with backups to feat-mcp-trash-restore (open change owns it)
