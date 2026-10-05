# Tasks: fix-backup-git-path

- [x] 1.1 `path = [ pkgs.git ]` в pm-serve
  **DoD:** `nix flake check` зелен; в сгенеринном юните есть `PATH=.*git`
  **Файлы:** `flake.nix`
  **Результат:** check all passed; юнит-проверка переносится на хост-свитч (1.2)

- [ ] 1.2 Валидация, PR, архив
  **DoD:** Forgejo run success; хост-фект: после свитча в логе нет
  `backup degraded`, `which git` виден из сервиса
