# Tasks: fix-backup-ssh-path

- [x] 1.1 openssh в path сервиса
  **DoD:** `nix flake check` зелен
  **Файлы:** `flake.nix`
  **Результат:** ниже

- [ ] 1.2 Валидация, PR, архив
  **DoD:** Forgejo run success; хост-фект: после свитча пуш уходит
  (коммиты видны в pm-data), в логе нет `cannot run ssh`
