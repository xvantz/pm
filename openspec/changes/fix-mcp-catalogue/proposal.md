# Proposal: fix-mcp-catalogue

## Why

`mcp-server` каталог говорит "exactly 14 tools", в коде 16. pm-close добавил тулы со своими дельтами, счетчик никто не тронул. Править число под каждый тул это toil: каталог должен быть выводимым, а не перечисленным.

## What Changes

- MODIFIED `Tool catalogue`: сервер отдает зарегистрированный набор (`RegisterPMTools`), live `tools/list` и есть каталог. Никаких чисел в спеке. Инварианты (непусто, уникальные имена, `type: object`) asserts тест.
- Тест: проверка уникальности имен в `TestToolsList_AllSchemasAreObjects` (схемы уже покрыты для всех зарегистрированных тулов).

## Impact

- Affected specs: `mcp-server` (MODIFIED: 1 requirement, сценарии сохранены + 1 новый).
- Affected code: `internal/mcp/mcp_test.go` (5 строк), спека.
- trash-restore позже добавит тулы без правок каталога. Проверка этого и есть DoD правила.
