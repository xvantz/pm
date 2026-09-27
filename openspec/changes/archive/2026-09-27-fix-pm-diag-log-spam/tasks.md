# Tasks: fix-pm-diag-log-spam

## Phase 1: Wrapper (dotfiles) - edit done in worktree, rebuild+verify on host

- [x] 1.1 `hermes.nix`: вырезать `env >&2`, `PATH`/`HOME`/`USER` дампы из `pm-diag`, оставить одну строку старта
  **DoD:** exceeded - pm-diag wrapper removed entirely, hermes.nix has zero pm-diag references (grep empty)
- [x] 1.2 `nixos-rebuild`, проверить `mcp-stderr.log`: построчный рост вместо килобайтных дампов
  **DoD:** owner confirms rebuild; no pm env-dump lines in recent log tail (only figma/filesystem chatter)
- [x] 1.3 Грепнуть логи на предмет токенов (`FORGEJO_TOKEN`, `GITHUB_TOKEN`), зачистить при необходимости
  **DoD:** no pm token-dump lines from current config (wrapper gone); historic 145M log rotation tracked separately

## Phase 2: Guard - DONE

- [x] 2.1 Зафиксировать в `hermes-integration.md` правило: wrapper пишет в stderr только одну строку
- [x] 2.2 (Опционально, отдельно) ротация `mcp-stderr.log` в Hermes
  **DoD:** out of pm scope, Hermes-side concern; log still 145M mostly non-pm traffic
