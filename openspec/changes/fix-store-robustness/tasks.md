# Tasks: fix-store-robustness

## Phase 1: Resolve + counter

- [ ] 1.1 `ResolveProject`: один скан, индекс number/id/prefix за проход
- [ ] 1.2 `next_number`: инкремент под flock создания проекта, тест на конкурентный `add_project` (N горутин - N уникальных номеров)
  - SUPERSEDED by `pm serve` (single writer): daemon mutex serializes creates,
    E2E доказал 20 параллельных созданий без дубликатов. Актуально только если
    вернется прямой доступ к файлам из нескольких процессов.
- [ ] 1.3 Бенч до/после на сторедже из 50 проектов (доказать выигрыш, а не верить)

## Phase 2: Corruption visibility

- [ ] 2.1 `ListProjects`: `unreadable` записи вместо silent skip (тип в `types.go`)
- [ ] 2.2 `doctor`: `--dir`/`PM_DIR` поддержка, секции counter/backups/corrupt, nonzero exit при проблемах
- [ ] 2.3 Тесты: битый YAML виден в list и doctor; MCP `list_projects` не падает на битом проекте

## Phase 3: Docs

- [ ] 3.1 README: поведение при битых данных, как чинить через trash/backup
