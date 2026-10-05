# Tasks: add-backup-token-auth

## Phase 1: Код

- [x] 1.1 Токен в gitbackup env
  `Config.Token`; заголовок только когда задан; без токена env чистый
  **DoD:** unit-тест env: с токеном несет `http.extraHeader`, без - нет
  **Файлы:** `internal/gitbackup/gitbackup.go`
  **Результат:** env через GIT_CONFIG_* (не `-c`, токен вне `ps`)

- [x] 1.2 Флаг/env в serve
  `--backup-token` + `PM_BACKUP_TOKEN`
  **DoD:** `pm serve --help` показывает флаг; `go build ./...` зелен
  **Файлы:** `internal/cli/serve.go`
  **Результат:** build зелен (флаг проверен в help вручную при ревью)

- [x] 1.3 Тесты пути с токеном
  Коммит+пуш в bare с заданным токеном (file-remote)
  **DoD:** `CGO_ENABLED=0 go test -count=1 ./internal/gitbackup/` зелен
  **Файлы:** `internal/gitbackup/gitbackup_test.go`
  **Результат:** TokenEnv + PushToBareWithToken зелены. Честно: HTTPS E2E
  локально не поднять - покрытие на уровне env + file-remote

## Phase 2: Nix и финал

- [x] 2.1 Опция `services.pm.backup.tokenFile` + проброс
  sops-конвенция, проброс как `PM_BACKUP_TOKEN` только когда задан
  **DoD:** `nix flake check` зелен
  **Файлы:** `flake.nix`
  **Результат:** check all passed; секрет читается в рантайме через
  `cat` (не в store); значение - путь при keyFile, контент при tokenFile

- [ ] 2.2 Валидация, PR, архив
  **DoD:** `validate --all --strict` 0 failed; Forgejo run success;
  `openspec list` без ченжа
