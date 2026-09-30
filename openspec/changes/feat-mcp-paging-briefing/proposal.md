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

**Пересечение файлов:** `internal/mcp/tools.go` делится с
`feat-mcp-trash-restore` (новые тулы) и `feat-close-consent` (схема
`close_project`). По правилу 3 AGENTS.md мерджится первым среди трёх — он трогает
и каталог тулов, и брифинг, то есть пересекается с обоими. Остальные ждут его
или идут в `git worktree`.
