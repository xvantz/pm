# Tasks: fix-nix-packaging

## Phase 1: Build

- [ ] 1.1 `nix build .#pm`, подставить настоящий `vendorHash` из ошибки `got:`
- [ ] 1.2 Проверить сборку с чистого клона (`git clean -fdx` эквивалент, пустой store-путь)
- [ ] 1.3 Добавить `--version` в `cmd/pm` и `cmd/pm-mcp`, проверить вывод `0.1.0`+rev

## Phase 2: Wiring

- [ ] 2.1 `flake.nix` модуль: `dataDir` пробрасывается в запуск (документировать контракт)
- [ ] 2.2 Dotfiles: `hermes.nix` использует `config.services.pm` (package + dataDir), убрать хардкод `--dir /data/pm`
- [ ] 2.3 `hermes-integration.md`: переписать под факт после wiring

## Phase 3: Verify

- [ ] 3.1 `nixos-rebuild`, `pm --version` и `pm-mcp --version` отвечают одинаково
- [ ] 3.2 MCP `pm` стартует с новым пакетом, данные на месте
