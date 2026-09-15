# Tasks: fix-mcp-transport-framing

## Phase 1: Spike (выбор варианта)

- [ ] 1.1 Подтвердить формат Hermes Python MCP клиента: NDJSON построчно (байты из живого handshake)
- [ ] 1.2 Оценить официальный MCP Go SDK: покрывает ли `tools/list`, `tools/call`, `ping`, string ID
- [ ] 1.3 Зафиксировать решение (A NDJSON-откат / B SDK) в `design.md` этого ченжа

## Phase 2: Implement

- [ ] 2.1 Переписать `internal/mcp/server.go` под выбранный вариант
- [ ] 2.2 Обновить `readMCPResponse` и все тесты в `mcp_test.go` под новый формат
- [ ] 2.3 Добавить interop-тест: золотая запись полного handshake байт-в-байт
- [ ] 2.4 `go test ./... -count=1`, `go vet ./...`

## Phase 3: Deploy (вместе с fix-mcp-list-projects-schema)

- [ ] 3.1 PR, merge, обновить flake input, `nixos-rebuild`
- [ ] 3.2 Живая проверка: `tools/list` и `tools/call list_projects` из Hermes, сервер не паркуется 30 минут
