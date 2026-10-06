# Tasks: fix-backup-service-path

- [x] 1.1 `path` на уровень сервиса
  **DoD:** `nix flake check` зелен
  **Файлы:** `flake.nix`
  **Результат:** check all passed

- [ ] 1.2 Валидация, PR, архив
  **DoD:** Forgejo run success; хост-фект: после свитча в логе юнита нет
  `Unknown key`, в логе демона нет `backup degraded`
