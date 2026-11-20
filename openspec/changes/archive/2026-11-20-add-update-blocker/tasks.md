# Tasks: add-update-blocker

- [x] 1.1 API принимает правку блокера
  **DoD:** `PATCH /api/projects/{ref}/steps/{step}/blockers/{blk}` меняет
  title и/или reason, id не двигается; пустой title → `400`; коллизия slug →
  `409`; неизвестный блокер → `404`; вызов без полей → `400`
  **Файлы:** `internal/api/server.go`, `internal/api/patch_blocker_test.go`
  **Результат:** 6 тестов `TestBlockerPatch_*` — pass

- [x] 1.2 `apistore` доносит правку до демона
  **DoD:** регресс-тест проверяет значение, прочитанное из демона, а не из
  возвращённой структуры; на старом коде падает
  **Файлы:** `internal/apistore/apistore.go`, `internal/client/client.go`,
  `internal/apistore/blocker_edit_test.go`
  **Результат:** `TestRemote_EditBlockerPropagates` — pass (title и reason
  меняются в демоне, id прежний)

- [x] 1.3 MCP-инструмент `update_blocker`
  **DoD:** `go test ./internal/mcp/` зелёный; `tools/list` содержит
  `update_blocker`; покрыты все сценарии дельты
  **Файлы:** `internal/mcp/tools.go`, `internal/mcp/update_blocker_test.go`
  **Результат:** 5 тестов `TestUpdateBlocker_*` — pass

- [x] 1.4 Живая проверка: настоящий демон + настоящий MCP по stdio
  **DoD:** собранные `pm-serve` и `pm-mcp` на временном сторе: `update_project`
  меняет title+status, `update_step` ренамит шаг, `update_blocker` меняет
  title+reason, вызов без полей отвергается, id шага и блокера не двигаются,
  а значения читаются обратно из демона
  **Файлы:** `/tmp/pm-serve`, `/tmp/pm-mcp`, скрипт прогона
  **Результат:** прогон зелёный: `tools/list` = 22 инструмента, все три
  правки применились и сохранились, повторное чтение через демон подтвердило

- [ ] 1.5 Валидация, PR, архив
  **DoD:** `validate --all --strict` 0 failed; `go vet ./...` и
  `CGO_ENABLED=0 go test ./...` зелёные; Forgejo run success
  **Файлы:** `openspec/specs/mcp-server/spec.md`,
  `openspec/specs/serve-api/spec.md`
