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

## Phase 2: Deploy (host, outside this repo)

- [ ] 2.1 systemd unit `pm-serve` + sops token (`PM_TOKEN`)
- [ ] 2.2 Smoke: `curl localhost:8472/healthz`, authed list_projects round-trip

## Phase 3: Migrate clients (separate changes)

- [ ] 3.1 CLI commands onto HTTP client (single code path)
- [ ] 3.2 pm-mcp onto HTTP client, drop bind-mount from hermes.nix
