# Tasks: feat-mcp-list-projects-filter (DoD обязателен)

## Phase 1: Схема и хендлер

- [x] 1.1 Схема `list_projects`: опциональный `status` с enum
  Добавить `status` (`active | completed | all`, default `active`) в `InputSchema` регистрации тула, `inputSchema.type == "object"` сохраняется.
  **DoD:** `go build ./...` зелен; `tools/list` отдает `status` в properties
  **Результат:** схема добавлена в `RegisterPMTools`, `type==object` на месте, build зелен.

- [x] 1.2 `handleListProjects`: фильтр плюс честные счетчики
  Фильтровать проекты стора по `status` в хендлере, стор не трогать. Ответ всегда несет `projects`, `count` (число возвращенных), `active_count`, `completed_count`, `total`; при `completed_count > 0` в default-ответе hint про `status=all`. Пустой `status` трактуется как `active`.
  **DoD:** ручной прогон на сторе с 6 active + 3 completed: default возвращает 6 записей и `completed_count: 3`; `status=all` возвращает 9
  **Результат:** реализовано в `handleListProjects`; фильтр в хендлере, стор и HTTP API не тронуты. Не-completed (paused/idea) считаются в active, чтобы `active+completed==total` всегда.

- [x] 1.3 Неизвестный статус - громкая ошибка
  Любое значение вне enum дает ошибку с перечислением допустимых (`active|completed|all`), а не молчаливый default.
  **DoD:** вызов со `status=foo` возвращает ошибку со словом `status` и списком допустимых
  **Результат:** `unknown status %q. Need: active|completed|all`, покрыто тестом.

- [x] 1.4 Описание тула как путеводитель
  Обновить `Description` `list_projects`: default active, когда звать `status=all` / `status=completed`.
  **DoD:** `tools/list` показывает новое описание; `validate --all --strict` зелен без правки требования `Tool catalogue`
  **Результат:** описание обновлено, снапшот `descriptions_test` зелен (ключи `Start here`, `get_project` на месте), validate `7 passed, 0 failed`.

## Phase 2: Доказательства

- [x] 2.1 Тест: default режет, но считает
  Фикстура со смешанными статусами: default возвращает только active, при этом `active_count + completed_count == total` и `count == len(projects)`.
  **DoD:** `go test ./internal/mcp/ -run TestListProjects_Filter -count=1` зелен; тест падает если убрать фильтр
  **Результат:** `TestListProjects_Filter_DefaultActiveCountsAll` зелен; проверяет суммы, отсутствие `completed` в default-выдаче и наличие `hint`.

- [x] 2.2 Тест: `status=all` и `status=completed`
  `all` возвращает все записи байт-совместимо с прежним поведением; `completed` возвращает только закрытые.
  **DoD:** оба подкейса зелены в том же `go test`
  **Результат:** `TestListProjects_Filter_AllAndCompleted` зелен.

- [x] 2.3 Тест: мусорный статус - ошибка
  `status=deleted` (или любой вне enum) дает ошибку, стор не читается молча.
  **DoD:** подкейс зелен; проверяет текст ошибки, а не подстроку в JSON
  **Результат:** `TestListProjects_Filter_UnknownStatus` зелен, требует `active|completed|all` в тексте ошибки.

## Phase 3: Документация и финал

- [x] 3.1 README: когда какой статус
  Короткий раздел "какой статус когда": аудит - default, история - `all`, разбор закрытого - `completed`.
  **DoD:** раздел существует, пример вызова с `status=all` присутствует
  **Результат:** обновлена строка таблицы тулов + абзац в `Reading through MCP`.

- [x] 3.2 `gofmt`, `vet`, полный тест, `validate --all --strict`
  **DoD:** `gofmt -l` пусто; `go vet ./...` чист; `go test ./... -count=1` зелен; `validate --all --strict` показывает `0 failed`
  **Результат:** все зелено: `gofmt` пусто, `vet` чист, все пакеты `ok`, validate `7 passed, 0 failed`.

- [ ] 3.3 PR плюс CI плюс архив последним коммитом
  Ветка уже `feat/mcp-list-projects-filter`, пуш в нее, PR, дождаться CI success, архив последним коммитом в том же PR.
  **DoD:** CI зелен; `openspec list` без `feat-mcp-list-projects-filter`; baseline несет дельту
  **Результат:** _заполняется при завершении_

## Границы

- Стор и HTTP API не меняем.
- Пагинацию не вводим.
- Trash не трогаем.
- Поля элементов списка не режем.
