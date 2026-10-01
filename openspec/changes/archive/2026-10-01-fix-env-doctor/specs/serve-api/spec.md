# serve-api Specification (delta)

## MODIFIED Requirements

### Requirement: Nix options

The flake module SHALL expose `services.pm.listenAddr`
(default `127.0.0.1:8472`) and `services.pm.environmentFile`
(sops env file with `PM_TOKEN`, same convention as `hermes_env`).
An enabled service without `environmentFile` fails the build via
`assertions` with exact fix instructions. `services.pm.user` is REQUIRED
(no default). When set, a
`pm-serve` systemd service is created: `restart=always`,
token via `EnvironmentFile` (never in nix store or unit text),
`PM_DIR` from `services.pm.dataDir`.
The module also provisions `PM_TOKEN` into login shells itself
(`programs.zsh/bash.interactiveShellInit` sourcing the same env file
at runtime): no hand edits to shell configs needed.
The module SHALL also provision `PM_API` derived from `listenAddr`,
so changing the address never orphans clients on the default.

#### Scenario: host enables daemon

- **WHEN** `services.pm = { enable = true; environmentFile = <pm_env path>; }`
- **THEN** after `nixos-rebuild`, `curl localhost:8472/healthz` answers
  and authed `/api/projects` round-trips
- **WHEN** `services.pm = { enable = true; }` without `environmentFile`
- **THEN** the build fails with an assertion naming the missing option

#### Scenario: custom address reaches clients

- **WHEN** `listenAddr` is changed from the default
- **THEN** shells and MCP pick the new address from provisioned `PM_API` with no hand edits

## ADDED Requirements

### Requirement: Daemon-side integrity check

`GET /api/doctor` SHALL return an integrity report over the daemon's own
store: project/step/blocker/decision counts, orphan dirs, unreadable files,
and legacy/unreadable timestamp counts. It SHALL require auth like the rest
of `/api`. The check runs inside the single writer, so it observes the same
state every client sees.

Rationale: a CLI-side file walk is a second reader beside the daemon, and
the two can disagree about which root they checked. Moving the walk into the
daemon removes the divergence class instead of instrumenting it.

#### Scenario: Healthy store reports counts

- **WHEN** `GET /api/doctor` answers on a healthy store
- **THEN** the body carries the entity counts with empty issue lists
