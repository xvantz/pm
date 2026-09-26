# Proposal: feat-pm-serve-daemon

## Problem

PM data lives as YAML files on a shared volume. Every consumer (host CLI,
Hermes container via bind-mount, future clients) reads and writes files
directly. Consequences:

- Concurrent writers can corrupt YAML (see fix-store-robustness: flock,
  corrupt visibility - treating symptoms).
- Every consumer needs the data path mounted: `dataDir` vs `containerDataDir`
  confusion, hardcoded `--dir /data/pm` in two places, desync without errors.
- No single place for validation: lifecycle rules live in CLI handlers only.

## Solution

`pm serve`: the `pm` binary gains a server mode. One daemon on the host owns
the YAML store (single writer). All clients (CLI, MCP, future) talk HTTP to
`127.0.0.1:8472` with a Bearer token. Containers need no volume mounts,
only address + token.

## Scope

Phase 1 (this change):
- `internal/api`: stdlib ServeMux, Bearer auth middleware, JSON CRUD mirroring
  the store interface + domain lifecycle validation + briefing.
- `pm serve --addr 127.0.0.1:8472 --token ...` (or `PM_TOKEN` env).
- Tests via httptest, README section.

Out of scope (follow-ups):
- `pm serve` systemd unit + sops token wiring in dotfiles.
- Migrating CLI commands onto the HTTP client (single code path).
- Migrating pm-mcp onto the HTTP client (drop bind-mount in hermes.nix).
- Tailscale exposure for remote clients (mobile is explicitly out of scope).

## Decisions

- TCP + Bearer token (not unix socket): network clients are the goal,
  a socket cannot reach them. See PM decision
  `daemon-api-pm-serve-single-writer-localhost-+-token`.
- Listen on localhost only by default. Never `0.0.0.0` to the world.
- `GET /healthz` unauthenticated (systemd/monitoring); everything under
  `/api/*` requires the token.
