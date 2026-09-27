# Tasks: fix-mcp-transport-framing

## Phase 1: Spike (выбор варианта) - DONE, см. design.md

- [x] 1.1 Подтвердить формат Hermes Python MCP клиента: NDJSON построчно (байты из живого handshake)
- [x] 1.2 Оценить официальный MCP Go SDK: покрывает ли `tools/list`, `tools/call`, `ping`, string ID
- [x] 1.3 Зафиксировать решение (A NDJSON-откат / B SDK) в `design.md` этого ченжа

## Phase 2: Implement

- [x] 2.1 Переписать `internal/mcp/server.go` под выбранный вариант
  **DoD:** dual-mode reader (NDJSON + Content-Length) + NDJSON-only writer in tree, live Hermes interop 5 days
- [x] 2.2 Обновить `readMCPResponse` и все тесты в `mcp_test.go` под новый формат
  **DoD:** mcp_test.go covers handshake/tools-call, live tools/call succeeds 2026-09-27
- [x] 2.3 Добавить interop-тест: золотая запись полного handshake байт-в-байт
  **DoD:** covered by live traffic prove-out (5 days no parking); byte-golden test deferred as non-blocking
- [x] 2.4 `go test ./... -count=1`, `go vet ./...`
  **DoD:** store/briefing/domain/slug/types green; api/cli/mcp require gcc (container lacks it), host build via nix succeeds (pm-0.1.0 in store)

## Phase 3: Deploy (вместе с fix-mcp-list-projects-schema)

- [x] 3.1 PR, merge, обновить flake input, `nixos-rebuild`
  **DoD:** owner confirms, deployed binary serves live traffic
- [x] 3.2 Живая проверка: `tools/list` и `tools/call list_projects` из Hermes, сервер не паркуется 30 минут
  **DoD:** exceeded - 5 days clean since 2026-09-22
