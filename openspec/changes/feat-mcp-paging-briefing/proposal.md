# Proposal: feat-mcp-paging-briefing

## Why

Два гэпа MCP поверхности (P9, P11 пула): `get_project` отдает весь дамп без пагинации (большие проекты раздувают контекст), string ID в JSON-RPC не поддерживаются (только int), брифинг вообще без capability спеки.

## What Changes

- Пагинация/truncate для больших проектов (лимиты + курсоры или компактный режим).
- String ID в MCP сервере.
- Новая capability `briefing`: что считает, секции, рекомендации.

## Impact

- Affected specs: `mcp-server` (MODIFIED: транспорт ID, пагинация), новая `briefing` capability.
- Affected code: `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/briefing`.
