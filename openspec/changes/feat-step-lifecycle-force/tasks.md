# Tasks: feat-step-lifecycle-force

## Phase 1: Domain

- [ ] 1.1 `internal/domain/step.go`: переход в `done` с `force=true` + запись причины
- [ ] 1.2 Автозапись decision `force-completed` с reason при force-закрытии
- [ ] 1.3 Юнит-тесты: force из `todo`/`in_progress` ок, обычный `done` не из `review` по-прежнему падает

## Phase 2: Surface

- [ ] 2.1 CLI `pm step done --force --reason`: флаг, help-текст
- [ ] 2.2 MCP `done_step`: опциональные `force` + `reason` в inputSchema
- [ ] 2.3 `go test ./...`, обновить README пример lifecycle
