# Tasks: feat-mcp-trash-restore

## Phase 1: Store (бэкапы + листинг + resolve)

- [x] 1.1 `FileStore`: бэкап в `_meta/backups/<unix>/` перед каждым
  `DeleteProject`/`DeleteStep`/`DeleteBlocker`/`DeleteDecision`, ротация 20
  **DoD:** тест на tmpdir: удаление шага пишет бэкап с тем же содержимым; 21
  удаление подряд оставляет ровно 20 каталогов бэкапов
- [x] 1.2 `TrashList` отдает структуры (trash-имя, id, number, title, время
  удаления); `TrashRestore` резолвит точное имя, номер или title;
  неоднозначность - ошибка со списком кандидатов; коллизия номера - отказ
  **DoD:** тесты: точное имя чинит; title двух удаленных с общим префиксом
  падает со списком кандидатов и ничего не восстанавливает; неизвестное имя
  падает с именем в ошибке
- [x] 1.3 Интерфейс + все реализации: `store.go`, `mock.go`, apistore, client,
  api-хендлеры, `cli/trash.go`, `cli_test.go` под новый тип
  **DoD:** `CGO_ENABLED=0 go build ./...` зелен; `go test ./internal/store/
  ./internal/cli/ -count=1` зелен

## Phase 2: MCP surface

- [x] 2.1 Тулы `trash_list` + `trash_restore` в `RegisterPMTools` со схемами
  `type: object`; `trash_clean` НЕ добавляем
  **DoD:** `tools/list` содержит оба тула и не содержит `trash_clean`;
  `TestToolsList_AllSchemasAreObjects` зелен
- [x] 2.2 MCP-тесты обоих тулов, включая ambiguity-путь
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1` зелен
- [x] 2.3 README: раздел про корзину, бэкапы и ручное восстановление; кто что
  может (MCP - корзина, CLI - все)
  **DoD:** раздел есть; процедура восстановления из `_meta/backups` описана
  шагами

## Phase 3: Merge + archive

- [x] 3.1 Полный прогон: build, vet, fmt, `validate --all --strict`
  **DoD:** все зелено, 0 failed
- [ ] 3.2 PR + CI + архив последним коммитом; P3 пула в `[x]`
  **DoD:** Forgejo run success; `openspec list` без `feat-mcp-trash-restore`;
  P3 со ссылкой на PR

## Границы

- Корзины для шагов нет, только бэкапы. Step-trash - отдельный ченж.
- `trash_clean` в MCP не отдаем. Агент не стирает безвозвратно.
- `pm doctor` бэкапы не трогает.
