# Tasks: fix-pm-diag-log-spam

## Phase 1: Wrapper (dotfiles) - edit done in worktree, rebuild+verify on host

- [x] 1.1 `hermes.nix`: вырезать `env >&2`, `PATH`/`HOME`/`USER` дампы из `pm-diag`, оставить одну строку старта
- [ ] 1.2 `nixos-rebuild`, проверить `mcp-stderr.log`: построчный рост вместо килобайтных дампов
- [ ] 1.3 Грепнуть логи на предмет токенов (`FORGEJO_TOKEN`, `GITHUB_TOKEN`), зачистить при необходимости

## Phase 2: Guard - DONE

- [x] 2.1 Зафиксировать в `hermes-integration.md` правило: wrapper пишет в stderr только одну строку
- [ ] 2.2 (Опционально, отдельно) ротация `mcp-stderr.log` в Hermes
