# Why

Прод-MCP лежит: Hermes паркует сервер `pm` каждые 5 минут с `ValidationError: 1 validation error for ListToolsResult` (см. `/data/.hermes/logs/errors.log`). Причина - `list_projects` единственный тул из 14 с `InputSchema = {}`, без `"type": "object"`. Pydantic-валидация Hermes режектит весь `tools/list` из-за одного тула. Один символ роняет все 14 тулов.

# What Changes

- `internal/mcp/tools.go`: `InputSchema` у `list_projects` меняется с `{}` на `{"type":"object","properties":{}}`.
- Добавляется regression-тест: сериализованный `tools/list` ответ валидируется на то, что каждый `inputSchema` содержит `"type": "object"`.
- Пересборка пакета, обновление flake input `pm` в dotfiles, `nixos-rebuild`.

# Impact

- Affected specs: `mcp-server` (MODIFIED: жесткое требование к схемам).
- Affected code: `internal/mcp/tools.go`, `internal/mcp/mcp_test.go`.
- Риск минимальный: меняется только схема в ответе `tools/list`, поведение хендлера то же. Откат - предыдущий rev пакета.
