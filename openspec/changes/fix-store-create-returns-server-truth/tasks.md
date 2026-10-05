# Tasks: fix-store-create-returns-server-truth

## Phase 1: Интерфейс и реализации

- [x] 1.1 `Store.CreateProject` в интерфейсе + deprecated-метки на
  `NextNumber`/`AdvanceNextNumber`
  **DoD:** `grep -n 'CreateProject' internal/store/store.go` не пуст;
  `CGO_ENABLED=0 go build ./...` показывает всех нарушителей списком (ожидаемо)
  **Файлы:** `internal/store/store.go`
  **Результат:** интерфейс + сентинелы ErrEmptyTitle/ErrBadID/ErrProjectExist;
  build перечислил 4 реализации, все покрыты в 1.2

- [x] 1.2 Три реализации: FileStore (счетчик как в `server.go:174-190`,
  спускается внутрь), MockStore (max+1), apistore (возвращает ответ
  `client.CreateProject` вместо `_`)
  **DoD:** `CGO_ENABLED=0 go build ./...` зелен
  **Файлы:** `internal/store/filestore.go`, `internal/store/mock.go`,
  `internal/apistore/apistore.go`
  **Результат:** build зелен; коммент apistore переписан на новый контракт

## Phase 2: Миграция caller

- [x] 2.1 CLI и MCP зовут `CreateProject`, re-read уходит
  Поведение то же: заголовок `Project #N` с серверным номером, UUID в `Next:`
  **DoD:** `grep -n 'NextNumber' internal/cli/project.go internal/mcp/tools.go` пуст;
  trash-gap тест зелен без перечитки в хендлере
  **Файлы:** `internal/cli/project.go`, `internal/mcp/tools.go`,
  `internal/mcp/add_project_stale_test.go` (без изменений или с упрощением)
  **Результат:** оба без NextNumber; дабл переписан на CreateProject-override,
  тест зелен. Побочка: MCP раньше ставил StatusActive, теперь везде StatusIdea
  как у сервера и CLI - унификация, было расхождение

- [x] 2.2 Серверный `handleProjectCreate` зовет стоpовый `CreateProject`
  вместо ручного NextNumber/Advance/Save (одна правда о счетчике)
  **DoD:** `grep -n 'NextNumber\|AdvanceNextNumber' internal/api/server.go` пуст;
  `CGO_ENABLED=0 go test ./internal/api/` зелен
  **Файлы:** `internal/api/server.go`
  **Результат:** хендлер 12 строк, ошибки по сентинелам (400/409);
  TestProjectCreateID зелен без правок

- [x] 2.3 Удалить `NextNumber`/`AdvanceNextNumber` из интерфейса (если все
  caller мигрировали) или зафиксировать deprecated-статус с датой удаления
  **DoD:** решение записано в proposal Non-goals/Impact; интерфейс без мертвых методов
  **Файлы:** `internal/store/store.go`, тест-дублеры в `*_test.go` (компилятор перечислит)
  **Результат:** решение - оставить deprecated: методы еще нужны прямому
  файловому тулингу и тестам счетчика (store_test), creation-путь их не зовет

## Phase 3: Финал

- [ ] 3.1 Полный прогон + валидация + PR + архив
  **DoD:** `CGO_ENABLED=0 go test ./...` зелен; `validate --all --strict`
  0 failed; Forgejo run success; `openspec list` без ченжа
