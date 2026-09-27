# Tasks: fix-nix-packaging

## Phase 1: Build

- [x] 1.1 `nix build .#pm`, подставить настоящий `vendorHash` из ошибки `got:` (только на хосте, в контейнере нет nix)
  **DoD:** flake.nix carries sha256-4HpSNp8hmMkcMeO9SGuq8hqQq6tvU1mJO2Nfvt50amQ=, store path pm-0.1.0 exists (errors.log)
- [x] 1.2 Проверить сборку с чистого клона (`git clean -fdx` эквивалент, пустой store-путь)
  **DoD:** host build succeeded per owner, package deployed
- [x] 1.3 Добавить `--version` в `cmd/pm` и `cmd/pm-mcp`, проверить вывод `0.1.0`+rev

## Phase 2: Wiring

- [x] 2.1 `flake.nix` модуль: `dataDir` пробрасывается в запуск (документировать контракт)
- [x] 2.2 Dotfiles: `hermes.nix` использует `config.services.pm` (package + dataDir), убрать хардкод `--dir /data/pm`
  **DoD:** hermes.nix pm block references services.pm.package, no --dir hardcode
- [x] 2.3 `hermes-integration.md`: переписать под факт после wiring

## Phase 3: Verify

- [x] 3.1 `nixos-rebuild`, `pm --version` и `pm-mcp --version` отвечают одинаково
  **DoD:** owner confirms rebuild, version flags in tree (cmd/pm-mcp/main.go --version)
- [x] 3.2 MCP `pm` стартует с новым пакетом, данные на месте
  **DoD:** live MCP pm tools serve real projects 2026-09-27
