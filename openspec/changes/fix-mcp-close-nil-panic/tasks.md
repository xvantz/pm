# Tasks: fix-mcp-close-nil-panic

## Phase 1: Фикс

- [x] 1.1 Сервер: confirm возвращает и проект, и план
  `ClosePlan` до закрытия под `s.mu`; комментарий "never both" переписать
  **DoD:** `grep -n 'never both' internal/api/server.go` пуст
  **Файлы:** `internal/api/server.go`
  **Результат:** confirm возвращает {Project, Plan}, комментарий never both переписан

- [x] 1.2 Хендлер: nil-guard на confirm-пути
  План nil (любой сторонний стор) - сообщение без счетчика, не паника
  **DoD:** тест с планом nil возвращает успех вместо паники
  **Файлы:** `internal/mcp/tools.go`
  **Результат:** nil-plan дает сообщение без счетчика, не панику

## Phase 2: Регресс и финал

- [x] 2.1 Регресс-тест через httptest-демона
  confirm `close_project` через apistore поверх тестового сервера:
  на текущем коде паника, после фикса зелен + счетчик в тексте
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1 -run CloseDaemon -v` зелен;
  временный откат серверного фикса роняет тест (доказать revert-проверкой)
  **Файлы:** `internal/mcp/close_daemon_test.go`
  **Результат:** оба зелены; revert-проверка: старый контракт роняет daemon-тест, guard держит. E2E: процесс жив после confirm, счетчик в тексте

- [ ] 2.2 Полный прогон + PR + архив
  build, vet, fmt, `validate --all --strict`, Forgejo run success
  **DoD:** run success; `openspec list` без ченжа; baseline несет дельту

## Границы

- Флоу и тексты не меняем, только контракт ответа и guard.
