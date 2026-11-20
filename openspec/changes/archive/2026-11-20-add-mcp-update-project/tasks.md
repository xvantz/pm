# Tasks: add-mcp-update-project

- [x] 1.0 Пин версии openspec CLI
  **DoD:** `validate --all --strict` даёт 0 failed в CI на PR; версия
  `1.9.0` стоит в `.forgejo/workflows/ci.yml` и `scripts/hooks/pre-commit`;
  доказательство, что виноват был `@latest`: на 2.1.x тот же baseline даёт
  5 failed
  **Файлы:** `.forgejo/workflows/ci.yml`, `scripts/hooks/pre-commit`
  **Результат:** run 87 (PR #44) — CI success на 1.9.0; на 2.1.0/2.1.1
  baseline даёт 5 failed

- [x] 1.1 API принимает `title` в `PATCH /api/projects/{ref}`
  **DoD:** `go test ./internal/api/` зелёный; запрос с `{"title":""}` даёт
  `400`, а `GET` после него отдаёт старый заголовок
  **Файлы:** `internal/api/server.go`, `internal/api/patch_title_test.go`
  **Результат:** `TestProjectPatch_TitleRenamesOnly`,
  `TestProjectPatch_EmptyTitleRefused` — pass

- [x] 1.2 Переименование шага в API
  **DoD:** `PATCH /api/projects/{ref}/steps/{step}` меняет только заголовок,
  id не двигается; пустой `title` → `400`; slug-коллизия → `409`;
  неизвестный шаг → `404`
  **Файлы:** `internal/api/server.go`, `internal/api/patch_title_test.go`
  **Результат:** четыре теста `TestStepPatch_*` — pass

- [x] 1.3 `apistore` доносит title до демона (проект и шаг)
  **DoD:** регресс-тесты падают на старом коде и зеленеют на новом: со
  снятым `&p.Title` тест даёт `title = "Old Name", want "New Name"`
  **Файлы:** `internal/apistore/apistore.go`,
  `internal/apistore/rename_test.go`, `internal/client/client.go`
  **Результат:** `TestRemote_RenameProjectPropagates`,
  `TestRemote_RenameStepPropagates` — pass на новом коде и fail на старом
  (проверено временным откатом строки)

- [x] 1.4 MCP-инструменты `update_project` и `update_step`
  **DoD:** `go test ./internal/mcp/` зелёный; `tools/list` содержит оба
  инструмента; все сценарии из дельт покрыты
  **Файлы:** `internal/mcp/tools.go`,
  `internal/mcp/update_project_test.go`, `internal/mcp/update_step_test.go`,
  `internal/mcp/mcp_test.go`
  **Результат:** 8 тестов `update_project`, 4 теста `update_step` — pass;
  жёсткий счётчик инструментов в `TestRegisterPMTools` убран (спека прямо
  запрещает хардкодить количество: живой `tools/list` и есть каталог)

- [ ] 1.5 Проверка в живом MCP
  **DoD:** в текущей сессии `update_project` переименовывает проект #17 и
  возвращает имя обратно (проверено `list_projects`); `update_step`
  переименовывает шаг и оставляет id; вызов без полей отдаёт ошибку
  **Файлы:** пересборка `pm-mcp` + `pm-serve` на хосте

- [ ] 1.6 Валидация, PR, архив
  **DoD:** `validate --all --strict` 0 failed; `go vet ./...` и
  `CGO_ENABLED=0 go test ./...` зелёные; Forgejo run success; спеки
  `mcp-server` и `serve-api` обновлены в baseline, изменение в
  `openspec/changes/archive/`
  **Файлы:** `openspec/specs/mcp-server/spec.md`,
  `openspec/specs/serve-api/spec.md`
