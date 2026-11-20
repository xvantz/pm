# Tasks: add-mcp-update-project

- [ ] 1.1 API принимает `title` в `PATCH /api/projects/{ref}`
  **DoD:** `go test ./internal/api/` зелёный; новый тест
  `TestProjectPatch_Title` проходит; запрос с `{"title":""}` даёт `400`,
  а `GET` после него отдаёт старый заголовок
  **Файлы:** `internal/api/server.go`, `internal/api/patch_title_test.go`

- [ ] 1.2 `apistore` отправляет `title` в теле PATCH
  **DoD:** `go test ./internal/apistore/` зелёный; тест проверяет тело
  запроса (`title` есть, `tags: []` отправляется), а не только ответ
  **Файлы:** `internal/apistore/apistore.go`, `internal/apistore/apistore_test.go`

- [ ] 1.3 MCP-инструмент `update_project`
  **DoD:** `go test ./internal/mcp/` зелёный; `tools/list` содержит
  `update_project` со схемой-объектом; все сценарии из delta-спеки
  покрыты тестами `internal/mcp/update_project_test.go`
  **Файлы:** `internal/mcp/tools.go`, `internal/mcp/update_project_test.go`

- [ ] 1.4 Проверка в живом MCP
  **DoD:** в текущей сессии `update_project` переименовывает проект #17
  (проверено `list_projects`), затем возвращает исходное имя; статус
  переключается туда и обратно; `update_project` без полей отдаёт ошибку
  **Файлы:** пересборка `pm-mcp` + `pm-serve` на хосте

- [ ] 1.5 Валидация, PR, архив
  **DoD:** `validate --all --strict` 0 failed; `go vet ./...` и
  `CGO_ENABLED=0 go test ./...` зелёные; Forgejo run success; спека
  `mcp-server` и `serve-api` обновлены в baseline, изменение в
  `openspec/changes/archive/`
  **Файлы:** `openspec/specs/mcp-server/spec.md`,
  `openspec/specs/serve-api/spec.md`
