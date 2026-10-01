# Proposal: chore-mcp-tool-descriptions

## Why

У каталога 19 тулов, но описания двух сортов. Три read-тула (`get_project`,
`list_steps`, `close_project`) уже говорят агенту куда идти дальше - их писали
в bounded-reads. Остальные 15 - одна строка без маршрута: `done_step` "Mark a
step as done", `resolve_blocker` "Resolve a blocker on a step". Агент не знает
что до, что после, сколько стоит вызов, и жжет лишние `tools/call` на ощупь.

Исследование best practices (2026-10-01):

- Merge.dev "MCP tool descriptions" (https://www.merge.dev/blog/mcp-tool-description):
  front-load главного (агент может не дочитать), 1-2 предложения глагол+ресурс,
  workflow-предшественник прямо в описании (пример Salesforce
  `create_contact`: "Call discover_required_fields first..."), разведение с
  соседом ("use search_calls_extensive instead" когда фильтр по юзеру),
  операционные детали (пагинация, лимиты) - в схему, не в описание.
- Anthropic "Writing Effective Tools for Agents"
  (https://modelcontextprotocol.info/docs/tutorials/writing-effective-tools/):
  писать для агента как новичку в команде (неявное сделать явным), unambiguous
  имена параметров (`user_id`, не `user`), консолидировать цепочки,
  high-signal ответы, actionable ошибки, detailed+concise режимы.

Наш случай ложится на оба источника один в один: уровни чтения (concise+detail)
уже есть, осталось дописать описания как навигацию.

## What Changes

Паттерн описания для каждого тула: **Что. Предшественник/следующий. Когда НЕ
звать (зови X вместо).** 1-2 предложения, главное первым. Только строки в
`internal/mcp/tools.go`, поведения ноль.

Черновики новых описаний (финал в коде, суть здесь):

- `list_projects` - "List all projects with status and progress. Start here to
  find a project_number, then read with get_project."
- `get_briefing` - "Daily briefing across projects: what changed, what is
  blocked, what to do next. For one project's state use get_project instead."
- `add_project` - "Create a project (status idea). Then add steps with
  add_step; for finished work closing use close_project, not delete."
- `add_step` - "Add a step (starts todo). Advance with
  start_step -> review_step -> done_step."
- `start_step` - "Begin work on a todo step. Next: review_step when done
  working, not done_step directly."
- `review_step` - "Mark work complete on an in_progress step; a human approves.
  Next: done_step. Blocked instead? Use add_blocker."
- `done_step` - "Mark a review step done. Fails unless in review - call
  review_step first. To finish everything at once use close_project."
- `add_blocker` - "Flag what blocks a step (step becomes blocked until
  resolved). Unblocks with resolve_blocker."
- `resolve_blocker` - "Unblock a step by blocker_id (see blocker_ids in
  list_steps or get_step). Step stays in its status, it does not advance."
- `add_decision` - "Record why something was decided. Read back with
  list_decisions; closing a project writes its own Closed decision."
- `list_decisions` - "List a project's recorded decisions. To record one use
  add_decision."
- `list_blockers` - "List unresolved blockers grouped by step (with reasons).
  Pass step_id for one step. Pair: list_steps shows blocker_ids cheaply,
  get_step shows one step's reasons in full."
- `delete_project` - "Move a project to trash (recoverable via trash_restore).
  For finished work prefer close_project, which records why."
- `trash_list` - "List trashed projects. Restore with trash_restore; there is
  no wipe tool here on purpose."
- `get_step` (добивка) - добавить "Needs blocker reasons for one step? This is
  the tool; list_steps carries only ids."

Плюс точечные добивки InputSchema там, где имя параметра врет: `step_id`
"Step slug/ID (see id in list_steps)" везде, `blocker_id` аналогично,
`target` у trash_restore уже хорош.

Уже хорошие (`get_project`, `list_steps`, `close_project`, `trash_restore`) не
трогаем кроме мелочей, если тест потребует единообразия.

## Non-goals

- Не добавляем тул `guide`/подсказчик: 20-й тул ради навигации - это tool
  sprawl, каталог и так висит в каждом `tools/list`. Навигация едет в
  описаниях, которые агент уже читает.
- Не вводим MCP-аннотации (`readOnlyHint`, `destructiveHint`): наш сервер их не
  отдает, допиливать `tools/list` ради клиентов, которые их может и не читают
  (читает ли Hermes - неизвестно), - отдельная работа с негарантированной
  пользой. Описания работают везде.
- Не меняем поведение, схемы типов, ответы хендлеров. Только человекочитаемые
  строки.
- Не переписываем README под это: дока для людей, описания для агентов.

## Impact

- Affected specs: `mcp-server` (MODIFIED: требование к описаниям как
  навигации + сценарии на каждый жизненный цикл).
- Affected code: `internal/mcp/tools.go` (только строки Description и пара
  описаний параметров), новый `internal/mcp/descriptions_test.go` (линт формы).
- Тест формы, не вкуса: каждое описание начинается с глагола, деструктивные
  упоминают путь назад/альтернативу, read-тулы упоминают уровень. Длина
  проверяется верхней границей, чтобы каталог не распух.
