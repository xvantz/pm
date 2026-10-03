# serve-api Specification

## Purpose
`pm serve` is the single-writer HTTP daemon owning the YAML store: CLI and MCP talk to it over localhost with a Bearer token instead of touching files, so concurrent writes serialize and validation lives in one place.

## Requirements

### Requirement: Single writer

The daemon SHALL be the only process touching the YAML store files.
All mutations go through HTTP handlers that hold a process-wide mutex.

#### Scenario: concurrent writes serialize

- WHEN two POSTs arrive simultaneously
- THEN both succeed, files contain both writes, no partial YAML

### Requirement: Bearer auth

Every request under `/api/*` SHALL present
`Authorization: Bearer <token>`. The token comes from `--token` flag
or `PM_TOKEN` env (flag wins). Comparison is constant-time.
Missing or wrong token yields `401` with JSON error body.
The token is provisioned everywhere it is needed: the daemon via sops
`pm_env`, the user shell and MCP from the same secret. No exceptions.

#### Scenario: no token

- WHEN `GET /api/projects` without header
- THEN `401 {"error":"unauthorized"}`

#### Scenario: health without auth

- WHEN `GET /healthz`
- THEN `200 {"status":"ok","version":"..."}` without any header

### Requirement: Remote-only clients

CLI commands and pm-mcp SHALL NOT touch YAML files (except `pm serve`,
`pm doctor`, and `pm init`, which are host-local by nature).
`openStore()` always returns the HTTP adapter; the address defaults to
the serve convention (`PM_API` overrides), token from `PM_TOKEN`.
`--dir` flags for data paths are removed; `pm serve --dir` stays
(the daemon owns the files).

#### Scenario: no daemon

- WHEN `pm project list` runs with nothing on 127.0.0.1:8472
- THEN a connection error names the address (no silent file fallback)

### Requirement: Endpoints mirror the store

Reads SHALL return the same JSON shapes as MCP read tools:

- `GET /api/projects` -> `[]Project`
- `POST /api/projects` `{title, goal?, tags?, id?}` -> `Project`
  (`id` pins a client UUID: 400 on garbage, 409 on collision; numbers stay server-assigned)
- `GET /api/projects/{ref}` -> `ProjectData` (ref = number or UUID)
- `PATCH /api/projects/{ref}` `{goal?, status?, tags?}` -> `Project`
- `DELETE /api/projects/{ref}` -> trash, `200 {"trashed": name}`
- `GET /api/projects/{ref}/steps` -> `[]Step`
- `POST /api/projects/{ref}/steps` `{title}` -> `Step` (slug id,
  duplicate title yields `409`)
- `POST /api/projects/{ref}/steps/{step}/{start,review,done}` ->
  `Step` with domain lifecycle validation (`422` on illegal transition)
- `DELETE /api/projects/{ref}/steps/{step}` -> `200`
- `GET /api/projects/{ref}/blockers` -> `[]Blocker`
- `POST /api/projects/{ref}/steps/{step}/blockers` `{title, reason?}`
  -> `Blocker` (default status `waiting`)
- `POST .../blockers/{blk}/resolve` -> `Blocker`
- `DELETE .../blockers/{blk}` -> `200`
- `GET /api/projects/{ref}/decisions` -> `[]Decision`
- `POST /api/projects/{ref}/decisions` `{title, reason?}` -> `Decision`
- `GET /api/briefing?date=&project=` -> briefing JSON

Errors are JSON `{"error": msg}` with codes:
`400` bad body, `401` auth, `404` unknown ref, `409` duplicate,
`422` lifecycle violation, `500` store failure.

#### Scenario: lifecycle violation surfaces as 422

- **WHEN** the client sends `done` for a step that is not in review
- **THEN** the server answers `422` with a JSON error body

### Requirement: CLI entry

`pm serve [--addr 127.0.0.1:8472] [--dir PATH] [--token ...]`
SHALL start the daemon. `--dir` overrides `PM_DIR`. Missing token is a
startup error telling the user to set `PM_TOKEN`.

#### Scenario: missing token fails fast

- **WHEN** `pm serve` starts with no `--token` and empty `PM_TOKEN`
- **THEN** it exits with an error naming the missing token

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

### Requirement: Close confirm carries the plan

`POST /api/projects/{ref}/close` with `confirm: true` SHALL return both the
closed project and the plan built before closing (`{confirmed: true, project,
plan}`), not the project alone. The plan lets every caller report how many
steps moved without a second read, and keeps file-backed and daemon-backed
stores observably identical: both answer confirm with a plan.

Rationale: the daemon answered confirm with `plan: nil` while `MockStore`
answered with a plan, so MCP crashed only in production (nil dereference in
`handleCloseProject`, SIGSEGV killing pm-mcp, found 2026-10-03). A contract
both sides share removes the divergence instead of guarding one caller.

#### Scenario: Confirm answers with both

- **WHEN** the client posts `close` with `confirm: true` and a reason
- **THEN** the response carries the closed project and the non-null plan,
  and the plan's step count matches the steps moved to done
