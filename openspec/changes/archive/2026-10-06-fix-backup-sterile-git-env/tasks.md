# Tasks: fix-backup-sterile-git-env

- [x] 1.1 Стерильный env для git-процессов
  **DoD:** новый тест зелен; полный suite зелен
  **Файлы:** `internal/gitbackup/gitbackup.go`,
  `internal/gitbackup/gitbackup_test.go`
  **Результат:** NOSYSTEM + GLOBAL=/dev/null; дубли в env проверены -
  побеждает последнее; регресс-тест GlobalInsteadOfIgnored зелен

- [ ] 1.2 Валидация, PR, архив
  **DoD:** `validate --all --strict` 0 failed; Forgejo run success;
  хост-фект: пуш уходит по HTTPS (коммиты в pm-data), в логе нет ssh-ошибок
