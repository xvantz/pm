# Tasks: feat-daemon-git-backup

## Phase 1: Git-обертка

- [x] 1.1 Пакет обертки: init/adopt/verify-remote/commit-all/async-push
  exec git, ключ через `GIT_SSH_COMMAND` с `IdentitiesOnly`, таймауты на
  сетевые вызовы, секреты никогда в логах
  **DoD:** `CGO_ENABLED=0 go build ./...` зелен; unit-тесты 1.2 зелены
  **Файлы:** новый `internal/gitbackup/gitbackup.go`
  **Результат:** пакет готов; disabled-конфиг = полный no-op без бранчей
  у caller; пуш `HEAD:main`, ретраи 5x10s, fail-closed remote

- [x] 1.2 Unit-тесты обертки
  Коммит на запись во временном репо; пуш в локальный bare через file-remote
  (ключ не нужен); remote-mismatch; сломанный гит не роняет вызов
  **DoD:** `CGO_ENABLED=0 go test -count=1 ./internal/gitbackup/ -run GitBackup -v` зелен;
  revert-проверка: пустой message или синхронный пуш роняют соответствующий тест
  **Файлы:** новый `internal/gitbackup/gitbackup_test.go`
  **Результат:** 7 тестов зелены; revert (убран add -A) роняет CommitOnAction;
  gofmt чист

## Phase 2: Хуки и опции

- [x] 2.1 Хуки в мутирующих хендлерах сервера
  Один коммит на действие с message `action <target>`; чтения не коммитят
  **DoD:** action через httptest-демона дает ровно один коммит с именем
  действия в `git log --oneline`; `TestProjectCreateID` и соседи зелены
  **Файлы:** `internal/api/server.go`
  **Результат:** 14 хуков + helper commit (warn, не ошибка); новый
  backup_test: create+step дают ровно 2 коммита; полный suite зелен.
  По ходу пойман баг: усыновленному репо identity не выставлялась -
  теперь pinIdentity на обоих путях, коммиты не зависят от global gitconfig.
  E2E на живом демоне (порт 8479, bare file-remote): create+step дали
  `add_project <uuid>` / `add_step <uuid> <slug>` локально и в bare,
  автор pm-serve; мертвый remote: API 201, коммит локально, warn+ретраи
  в фоне. Следы зачищены (/tmp)

- [x] 2.2 Nix-опции `services.pm.backup`
  `enable`, `repoUrl`, `keyFile`; проброс в сервис, assertions на пару
  (включено без repoUrl = ошибка сборки)
  **DoD:** `nix flake check` зелен; включение без repoUrl роняет evaluation
  с понятным message
  **Файлы:** `flake.nix`
  **Результат:** `nix flake check` all checks passed (модуль в полном
  NixOS-контексте); assertion зеркалит существующий environmentFile.
  Финальное доказательство - host `nixos-rebuild dry-activate` (за Ivan)

## Phase 3: Финал

- [ ] 3.1 Валидация, PR, архив
  **DoD:** `validate --all --strict` 0 failed; Forgejo run success;
  `openspec list` без ченжа; baseline несет `store-git-backup`
