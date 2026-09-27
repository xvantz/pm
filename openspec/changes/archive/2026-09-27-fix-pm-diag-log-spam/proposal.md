# Why

`mcp-stderr.log` распух до 147MB. Причина - `pm-diag` wrapper в `hermes.nix` дампит весь `env` (включая токены) и `PATH` в stderr при каждом старте MCP-сервера, а Hermes перезапускает сервер регулярно. Это жрет диск, замусоривает логи и светит секреты в plaintext-логах. Нарушает собственное требование stderr discipline из спеки `hermes-integration`.

# What Changes

- `hermes.nix` (dotfiles, кросс-репозиторий): из `pm-diag` убрать `env >&2` и `PATH` дампы, оставить одну строку `starting pm-mcp <version>` в stderr.
- Добавить ротацию/лимит размера `mcp-stderr.log` на стороне Hermes (отдельная задача, здесь только контракт: wrapper пишет мало).
- Проверить, что `FORGEJO_TOKEN`/`GITHUB_TOKEN` больше не попадают в логи.

# Impact

- Affected specs: `hermes-integration` (ADDED: лимит болтливости wrapper).
- Affected code: `/dotfiles/modules/system/hermes/hermes.nix` (не этот репозиторий - задача кросс-репозиторная, PR в dotfiles).
- Риск нулевой: меняется только stderr-диагностика, stdout-протокол не трогается.
