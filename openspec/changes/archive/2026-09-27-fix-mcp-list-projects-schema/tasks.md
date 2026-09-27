# Tasks: fix-mcp-list-projects-schema

## Phase 1: Fix - DONE on branch fix/mcp-schema-transport

- [x] 1.1 `internal/mcp/tools.go`: заменить `InputSchema` у `list_projects` на `{"type":"object","properties":{}}`
- [x] 1.2 `internal/mcp/mcp_test.go`: добавить тест, проверяющий `"type": "object"` у всех тулов в `tools/list`
- [x] 1.3 Прогнать `go test ./...` и `go vet ./...`

## Phase 2: Deploy

- [x] 2.1 Закоммитить в `xvantz/pm`, запушить (не в main напрямую - через PR по правилам)
  **DoD:** fix present in tree, hermes.nix consumes services.pm.package build pm-0.1.0 (store path in errors.log 2026-09-22)
- [x] 2.2 Обновить flake input `pm` в dotfiles, `nixos-rebuild`
  **DoD:** owner confirms rebuild done, live MCP pm tools answer
- [x] 2.3 Проверить `errors.log`: отсутствие `ValidationError` для `pm` в течение 15 минут, тул `mcp_pm_list_projects` доступен в Hermes
  **DoD:** zero pm ValidationError since 2026-09-22 15:09 (5 days clean), tools answer live 2026-09-27
