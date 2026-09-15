# Why

Nix-упаковка `pm` держится на честном слове. Три факта: (1) `flake.nix:41` до сих пор содержит `vendorHash = "sha256-AAA..."` плейсхолдер - локальный `nix build .#pm` падает, собирается только залоченный старый rev из кэша; (2) `services.pm.dataDir` (дефолт `/home/xvantz/Documents/pm`) никуда не пробрасывается - `hermes.nix` хардкодит `--dir /data/pm`, итого три источника правды о пути данных (модуль, hermes.nix, `PM_DIR`); (3) `pm-mcp --help` не показывает версию, хотя ldflags ее вшивают - задеплоенный rev неопознаваем с хоста. Любое обновление пакета сейчас - ручная хирургия.

# What Changes

- `flake.nix`: настоящий `vendorHash` (получить через `nix build` и подставить из `got:`).
- Модуль `services.pm`: `dataDir` реально используется - Hermes запускает `pm-mcp --dir ${cfg.dataDir}`, дефолт выравнивается на фактически смонтированный путь; `PM_DIR` остается оверрайдом для dev.
- `cmd/pm/main.go` + `cmd/pm-mcp/main.go`: `--version` выводящий `Version` из ldflags.
- `hermes-integration.md` приводится к факту.

# Impact

- Affected specs: `hermes-integration` (MODIFIED: упаковка и запуск).
- Affected code: `flake.nix`, `cmd/pm/main.go`, `cmd/pm-mcp/main.go`, dotfiles `services.nix`/`hermes.nix` (кросс-репо часть).
- После этого ченжа `nix build .#pm` работает с чистого клона - необходимое условие для всех остальных деплоев.
