# Tasks: fix-mcp-add-project-stale-number

## Phase 1: Фикс

- [x] 1.1 MCP хендлер: re-read после save + UUID в Next
  `handleAddProject`: после `SaveProject` читать по UUID (`GetProject`,
  fallback `ResolveProject`), печатать вычитанный номер; `Next: add_step
  {project_id: "<uuid>"}`; ошибка чтения - громкая, не вранье
  **DoD:** trash-gap тест печатает #14 вместо advisory #7
  **Файлы:** `internal/mcp/tools.go`
  **Результат:** re-read + UUID в Next; revert-проверка роняет тест (#7 вместо #14)

- [x] 1.2 apistore: переписать stale-комментарий
  Было "concurrent creates may print a stale number", стало - caller обязан
  перечитывать; оба caller (MCP + CLI) это делают
  **DoD:** `grep -n 'may print' internal/apistore/apistore.go` пуст
  **Файлы:** `internal/apistore/apistore.go`
  **Результат:** комментарий описывает re-read контракт

## Phase 2: Регресс и финал

- [x] 2.1 Регресс-тест trash-gap
  `trashGapStore` поверх MockStore (живой max 6, счетчик 14): печать #14,
  `add_step` по напечатанному UUID с первого раза, второй проект #15 тоже ок
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1 -run TestHandleAddProject_TrashGap -v` зелен; временный откат фикса роняет тест
  **Файлы:** `internal/mcp/add_project_stale_test.go`
  **Результат:** revert роняет (печатает #7), фикс зелен; `add_step` по UUID ок

- [x] 2.2 Полный прогон затронутых пакетов
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ ./internal/apistore/ ./internal/api/` зелен
  **Результат:** все три ok (2026-10-05)

- [ ] 2.3 Валидация, коммит, PR, архив
  `openspec validate --all --strict`, коммит кода + ченжа, PR, CI зелен,
  архив последним коммитом в том же PR
  **DoD:** `openspec list` без ченжа; baseline `mcp-server` несет дельту;
  Forgejo run success
