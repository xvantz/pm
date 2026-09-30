# Proposal: feat-mcp-blocker-refs

## Why

`list_steps` после bounded-reads отдает кратко: id, title, status, updated_at.
Шаг в статусе `blocked` выглядит так же, как `todo` - тот же набор полей.
Агент видит `blocked` и не знает, чем именно заблокировано, пока не вызовет
`get_step` или `list_blockers` вслепую.

Замерено раньше: `get_step` стоит 485 B и уже несет блокеры с `reason`.
Проблема не в весе, а в навигации (navigation - поиск пути): второй уровень
не подсказывает, куда идти дальше.

Отдельный `get_blocker` отклонен: блокеры живут внутри шага (обычно 0-2 на
шаг), `get_step` уже отдает их с причиной. Новый тул добавил бы 18-ю запись
в каталог ради экономии ~200 B. Дороже держать, чем вызывать лишний раз.

## What Changes

Без новых тулов, два расширения существующих:

- `list_steps`: каждый шаг несет `blocker_ids: []string`. Пусто = нет
  блокеров или шаг не заблокирован. Агент сразу видит связку шаг -> блокер
  и знает, что звать дальше (`get_step` или `resolve_blocker` уже требуют
  `step_id` + `blocker_id`).
- `list_blockers`: опциональный `step_id`. Без него - как раньше, все блокеры
  проекта группами по шагам. Со `step_id` - только блокеры этого шага.
  Backward compatible (обратно совместимый): старые вызовы без `step_id`
  отвечают как раньше.

## Impact

- Affected specs: `mcp-server` (MODIFIED: форма `list_steps`, фильтр
  `list_blockers`).
- Affected code: `internal/mcp/tools.go` - `jsonStepBrief`, `handleListSteps`,
  `handleListBlockers`, описания тулов. HTTP API демона не трогаем: форма
  ответа - свойство MCP-поверхности.
- Соседи: baseline `mcp-server` уже требует, что `list_steps` НЕ несет
  blockers/artifacts. Дельта расширяет это требование списком id без объектов.
- Пересечение по файлам: `internal/mcp/tools.go` делит с
  `feat-mcp-trash-restore` (открыт). Правило 3 AGENTS.md: мердж
  последовательный, этот первый - он меньше и уже в работе.
