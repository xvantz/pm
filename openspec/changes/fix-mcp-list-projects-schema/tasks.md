# Tasks: fix-mcp-list-projects-schema

## Phase 1: Fix

- [ ] 1.1 `internal/mcp/tools.go`: заменить `InputSchema` у `list_projects` на `{"type":"object","properties":{}}`
- [ ] 1.2 `internal/mcp/mcp_test.go`: добавить тест, проверяющий `"type": "object"` у всех тулов в `tools/list`
- [ ] 1.3 Прогнать `go test ./...` и `go vet ./...`

## Phase 2: Deploy

- [ ] 2.1 Закоммитить в `xvantz/pm`, запушить (не в main напрямую - через PR по правилам)
- [ ] 2.2 Обновить flake input `pm` в dotfiles, `nixos-rebuild`
- [ ] 2.3 Проверить `errors.log`: отсутствие `ValidationError` для `pm` в течение 15 минут, тул `mcp_pm_list_projects` доступен в Hermes
