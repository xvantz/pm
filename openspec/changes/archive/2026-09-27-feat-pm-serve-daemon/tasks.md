# Tasks: feat-pm-serve-daemon

## Phase 1: Daemon

- [x] 1.1 `internal/api/server.go`: ServeMux, Bearer middleware (constant-time),
  mutex-guarded handlers, JSON error bodies
- [x] 1.2 Endpoints: projects CRUD + trash, steps + start/review/done with
  domain validation, blockers + resolve, decisions, briefing
- [x] 1.3 `internal/api/server_test.go`: auth (401/200), project create/get,
  step lifecycle incl 422, duplicate 409, healthz without auth
- [x] 1.4 `pm serve` CLI wiring: --addr/--dir/--token, PM_TOKEN env,
  graceful shutdown on SIGINT/SIGTERM
- [x] 1.5 `go build ./...`, `go vet ./...`, `go test ./... -count=1`
- [x] 1.6 README section: run, token, endpoint table

## Phase 2: Deploy (host)

- [x] 2.1 systemd unit `pm-serve` + sops token (`PM_TOKEN`)
  **DoD:** services.nix sets services.pm enable/user/dataDir/environmentFile + sops pm_env with restartUnits pm-serve.service
- [x] 2.2 Smoke: `curl localhost:8472/healthz`, authed list_projects round-trip
  **DoD:** MCP pm tools answer live 2026-09-27, zero parking/ValidationError since 2026-09-22 (errors.log)

## Phase 3: Migrate clients

- [x] 3.1 `internal/client`: typed HTTP client, все эндпоинты + trash + briefing
- [x] 3.2 `internal/apistore`: `store.Store` поверх HTTP (delta-маппинг в
  lifecycle endpoints, NextNumber advisory, Advance no-op)
- [x] 3.3 CLI: `openStore()` → remote при `PM_API`, briefing remote branch,
  doctor всегда локальный; тесты hermetic через TestMain
- [x] 3.4 `pm-mcp`: remote при `PM_API` без проверки папки
- [x] 3.5 E2E: CLI remote полный цикл 1:1, MCP remote tools/call, daemon trash endpoints
- [x] 3.6 Host: убрать bind-mount из hermes.nix, MCP через PM_API+PM_TOKEN
  **DoD:** hermes.nix pm block uses package bin + PM_API/PM_TOKEN env, extraVolumes has no pm data mount

## Phase 4: Остаток openspec (пересмотреть после демона)

- [x] store-robustness частично не нужен (гонок нет при одном писателе)
  **DoD:** decision `store-robustness закрыт` recorded 2026-09-27, delta rewritten to daemon reality
- [x] trash-restore, lifecycle-force - по мере надобности
  **DoD:** kept as separate open changes feat-mcp-trash-restore, feat-step-lifecycle-force
