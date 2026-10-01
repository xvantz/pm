# Proposal: chore-mcp-actionable-responses

## Why

Описания (#32) говорят агенту куда идти ДО вызова. Но что происходит ПОСЛЕ -
ответы и ошибки - до сих пор говорят на чужом языке. Замер по коду 2026-10-01
(`internal/mcp/tools.go`):

1. **Next-подсказки на CLI-языке, мертвые для MCP-агента.** Успешные ответы
   несут `Next: pm step start 1 x`, `Next: pm add step ...`,
   сводка `get_project` - `pm list_blockers 1, or pm blocker resolve`.
   Агент за MCP не имеет шелла с `pm` (у него 19 тулов, не CLI). Правильный
   следующий шаг для него - имя тула (`start_step`), а не шелл-команда.
   Подсказка есть, толку ноль.
2. **Ошибки сырые, без действия.** `invalid args: ...`, `step_id is required`,
   `step "x" already exists in project #N` - констатация без "сделай X".
   Исключение-образец уже есть: `domain.ValidateStepDone` -
   `step "x" is todo, must be in review before done` (говорит требуемое
   состояние). Остальные до него не дотягивают.
3. **`get_step` без хинта вообще.** Детальный уровень молчит куда дальше
   (назад к `list_steps`? к `resolve_blocker` при blocked?), хотя сводка и
   пишущие тулы хинты имеют.

Источники те же: Anthropic effective-tools (actionable errors с примером
правильного вызова; high-signal ответы) и Merge.dev (операционка в схеме,
workflow в тексте). Этот ченж - вторая половина той же работы: #32 размечает
вход, этот - выход.

## What Changes

Паттерн ответа: **Факт. Следующий тул по имени. При blocked - куда за
причиной.** Паттерн ошибки: **Что случилось. Какое состояние нужно. Какой
вызов чинит.**

Конкретно:

- `Hint` сводки + все `Next:` в пишущих тулах (`add_project`, `add_step`,
  `start/review/done_step`, `add/resolve_blocker`, `add_decision`,
  `close_project`, `delete/trash_restore`): CLI-команды заменить именами
  MCP-тулов с параметрами (`start_step {project_id, step_id}`,
  `list_blockers {project_id}`, ...). Текст остается человекочитаемым, но
  имя тула - точное.
- `get_step`: добавить `hint` (blocked - `resolve_blocker` + `list_blockers`
  за причиной; иначе следующий по жизненному циклу из статуса).
- Ошибки обернуть в actionable (цепочка `handle*` в `tools.go`, доменные
  тексты не трогаем):
  - done без review: уже хорошо из домена, поднять как есть + назвать
    чинящий вызов (`review_step`);
  - unknown step id: назвать шаг + `list_steps` как источник живых id;
  - `already exists` (step/blocker/decision): назвать конфликт + действие
    (переиспользуй / переименуй);
  - `step_id is required` / `invalid args`: назвать missing param + пример
    минимальных аргументов;
  - close confirm без reason: назвать `reason` + двухвызовный флоу.
- Тест `internal/mcp/actionable_responses_test.go`: каждый `Next`/`Hint`
  называет реально существующий тул (регресс на CLI-язык: grep `pm step`
  / `pm list_` в ответах пуст); каждая покрытая ошибка содержит глагол
  действия; негатив - старый текст роняет тест.

## Non-goals

- Не меняем коды ошибок MCP (`-32602`/`-32603`) и транспорт - только тексты.
- Не трогаем доменные тексты (`internal/domain`, `internal/store`): обертка
  на уровне `handle*`, домен остается чистым для CLI.
- Не добавляем новые поля в API/CLI ответы: хинт `get_step` живет только в
  MCP-слое, как `Hint` сводки сегодня.
- Код только после мержа #32: оба ченжа правят `tools.go`, реализация здесь
  стартует от main с влитым #32.

## Impact

- Affected specs: `mcp-server` (ADDED: требование actionable-ответов).
- Affected code: `internal/mcp/tools.go` (строки ответов + обертки ошибок),
  новый `internal/mcp/actionable_responses_test.go`.
- Зависимость: реализуется после мержа PR #32 (тот же файл).
