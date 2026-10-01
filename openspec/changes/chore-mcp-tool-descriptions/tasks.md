# Tasks: chore-mcp-tool-descriptions

## Phase 1: Описания

- [x] 1.1 Переписать Description 15 тулов по паттерну
  Что/предшественник-когда-не-звать; черновики в proposal
  **DoD:** `grep 'Mark a step as done (must be in review first)' internal/mcp/tools.go`
  пуст; каждое новое описание начинается с глагола
  **Файлы:** `internal/mcp/tools.go` (только строки)
  **Результат:** 15 описаний переписаны, старые однострочники отсутствуют (grep пуст)

- [x] 1.2 Добить InputSchema-описания параметров
  `step_id`/`blocker_id` с намеком где взять id; `target` уже хорош
  **DoD:** каждый `step_id`/`blocker_id` в схемах содержит `list_steps` или
  `get_step`
  **Файлы:** `internal/mcp/tools.go` (только строки схем)
  **Результат:** 14 project_id + step_id/blocker_id указывают источник id

## Phase 2: Тест формы и финал

- [x] 2.1 `internal/mcp/descriptions_test.go`: лint описаний
  Глагол первым словом; деструктивные (`done_step`, `close_project`,
  `delete_project`) упоминают альтернативу/путь назад; read-тулы упоминают
  соседний уровень; кап длины (например 300 символов), чтобы каталог не распух
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1` зелен; тест падает,
  если вернуть любое старое однострочное описание
  **Файлы:** `internal/mcp/descriptions_test.go`
  **Результат:** оба теста зелены; негатив доказан: возврат старого done_step роняет тест

- [ ] 2.2 Полный прогон + PR + архив
  build, vet, fmt, `validate --all --strict`, Forgejo run success
  **DoD:** run success; `openspec list` без ченжа; baseline несет дельту

## Границы

- Поведение хендлеров не трогаем: ни одного изменения вне строк.
- Новый тул не создаем; аннотации MCP не вводим (см. Non-goals в proposal).
