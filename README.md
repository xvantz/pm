# PM — Project Memory

Long-term project memory for humans and AI.

Stores project state, steps, blockers, and architectural decisions — as YAML files under Git. One daemon owns the files (single writer); CLI and MCP talk to it over HTTP. No shared volumes, no concurrent writes.

---

## Concept

PM solves context recovery: instead of re-reading chats, notes, and git logs, you open `pm briefing` — and in 30 seconds you know what is happening across projects, what changed, what blocks, what to do today.

## Architecture

```
cmd/              # entry points
├── pm/           # CLI (humans)
└── pm-mcp/       # MCP server (agents, over stdio)
internal/
├── api/          # `pm serve` daemon: stdlib ServeMux, Bearer auth, single writer
├── client/       # typed HTTP client for the daemon
├── apistore/     # store.Store over HTTP (CLI/MCP need zero handler changes)
├── cli/          # CLI handlers (remote-only, except serve/doctor/init)
├── mcp/          # JSON-RPC 2.0, NDJSON framing
├── domain/       # business logic (step lifecycle machine)
├── store/        # FileStore (daemon-side) + MockStore + interface
├── slug/         # slug generator
├── types/        # data models
└── briefing/     # daily digest engine
```

Rule: only the daemon touches YAML. Everyone else goes through HTTP.

## Install

### Nix flake

```nix
# flake.nix
{
  inputs.pm.url = "github:xvantz/pm";

  outputs = { pm, ... }: {
    nixosConfigurations.nixos = nixpkgs.lib.nixosSystem {
      modules = [
        pm.nixosModules.default
        { services.pm.enable = true; }
      ];
    };
  };
}
```

Build:

```bash
nix build github:xvantz/pm
# or
nix run github:xvantz/pm#pm -- project list
```

### Go (dev)

```bash
git clone https://git.827482.xyz/xvantz/pm ~/Documents/pm
cd ~/Documents/pm
go build -o ~/.local/bin/pm ./cmd/pm
go build -o ~/.local/bin/pm-mcp ./cmd/pm-mcp
```

## Run

Start the daemon once (host side, owns the YAML):

```bash
export PM_TOKEN=$(openssl rand -hex 32)
pm serve --addr 127.0.0.1:8472 &
```

Point clients at it (address defaults to the serve convention, token from `PM_TOKEN`):

```bash
export PM_API=http://127.0.0.1:8472 PM_TOKEN=...
pm project list     # goes to the daemon
pm briefing         # computed by the daemon
```

## Usage

```bash
pm init                                          # create a store (daemon side)
pm project create "AdGuard Home"                 # create a project
pm add step 1 "Set up Caddy"                     # add a step
pm step start 1 setup-caddy                      # start the step
pm step review 1 setup-caddy                     # send to review
pm step done 1 setup-caddy                       # finish (review only)
pm blocker add --reason "no budget" 1 setup-caddy "Buy router"
pm blocker resolve 1 setup-caddy router          # unblock
pm decision add --reason "one binary" 1 "Go as language"
pm project close 1 "shipped"                 # bulk close, reason required
pm doctor                                        # integrity check (host-local, reads files)
pm trash list                                    # trashed projects
pm trash restore <name>                          # restore
pm briefing                                      # daily digest
```

### MCP (for agents)

pm-mcp is a JSON-RPC 2.0 server over stdio with NDJSON framing. 13 tools:

| Tool | Description |
|-----------|----------|
| `list_projects` | list projects (JSON) |
| `get_project` | project summary: counts, open blockers, last step (`detail:true` for everything) |
| `get_step` | one step in full, with its blockers and artifacts |
| `add_project` | create a project |
| `add_step` | add a step |
| `start_step` | todo → in_progress |
| `review_step` | send to review |
| `done_step` | finish (review only) |
| `add_blocker` | add a blocker |
| `resolve_blocker` | resolve a blocker |
| `add_decision` | record a decision with rationale |
| `close_project` | bulk-close: requires consent (see below) |
| `delete_project` | move to trash (prefer close for finished work) |
| `get_briefing` | generate a digest |
| `list_steps` | project steps, briefly (id/title/status/updated_at/blocker_ids) |
| `list_blockers` | blockers (JSON), optional step_id filter |
| `list_decisions` | decisions (JSON) |

In remote mode pm-mcp needs no data directory at all — only `PM_API` + `PM_TOKEN`.

### Hermes Agent integration

```nix
# hermes.nix
{ config, ... }: {
  services.hermes-agent.mcpServers.pm = {
    command = "${config.services.pm.package}/bin/pm-mcp";
    env.PM_API = "http://127.0.0.1:8472";
    env.PM_TOKEN = "\${PM_TOKEN}";
  };
}
```

After `nixos-rebuild` all 13 tools are available with the `mcp_pm_` prefix.

## Daemon (`pm serve`)

```bash
curl localhost:8472/healthz
curl -H "Authorization: Bearer $PM_TOKEN" localhost:8472/api/projects
```

| Method | Path | What it does |
|-------|------|-----------|
| GET | /healthz | liveness, no auth |
| GET/POST | /api/projects | list / create `{title, goal?, tags?, id?}` |
| GET/PATCH/DELETE | /api/projects/{ref} | details / `{goal?, status?, tags?}` / trash |
| GET/POST | /api/projects/{ref}/steps | list / create `{title}` |
| POST | .../steps/{step}/{start,review,done} | validated lifecycle |
| DELETE | .../steps/{step} | delete a step |
| GET | /api/projects/{ref}/blockers | list blockers |
| POST | .../steps/{step}/blockers | create `{title, reason?}` |
| POST | .../blockers/{blk}/resolve | resolve a blocker |
| DELETE | .../blockers/{blk} | delete a blocker |
| GET/POST | /api/projects/{ref}/decisions | list / create `{title, reason?}` |
| DELETE | .../decisions/{dec} | delete a decision |
| GET/DELETE | /api/trash | list / empty the trash |
| POST | /api/trash/{name}/restore | restore from trash |
| GET | /api/briefing?date=&project= | briefing JSON |

Codes: 400 body, 401 auth, 404 ref, 409 duplicate, 422 lifecycle, 500 store.
Listens on localhost by default. Expose outward only via Tailscale, never 0.0.0.0.

Auth: every `/api/*` call needs `Authorization: Bearer $PM_TOKEN`.
The token is provisioned automatically: daemon via sops `pm_env`,
user shell and MCP from the same secret.

Client-provided project UUIDs are honored (CLI/MCP pre-assign one and print it
in confirmation texts); the server validates the format (400) and rejects
collisions (409). Step numbers stay server-assigned.

## Data model

```yaml
# <project-id>/project.yaml
id: "0196f1a2-..."
number: 1
title: "AdGuard Home"
goal: "Run a DNS server"
status: active          # idea | active | paused | completed
tags: ["infrastructure"]
created_at: "2026-06-10"
updated_at: "2026-06-14"
```

```yaml
# <project-id>/steps/<slug>.yaml
id: setup-caddy
title: "Set up Caddy"
status: done            # todo | in_progress | review | done | blocked
project_id: "0196f1a2-..."
blockers: []
created_at: "2026-06-13T09:12:44Z"
updated_at: "2026-06-13T17:40:02Z"
```

```yaml
# blockers live inside step.yaml:
blockers:
  - id: router
    title: "Buy router"
    reason: "No budget"
    status: waiting      # waiting | active | resolved
    project_id: "..."
    step_id: "configure-dns"
    created_at: "2026-06-10T08:00:00Z"
    updated_at: "2026-06-10T08:00:00Z"
```

```yaml
# <project-id>/decisions/<slug>.yaml
id: use-go
title: "Go as language"
reason: "One binary, no dependencies"
date: "2026-06-13T11:05:19Z"
project_id: "..."
```

## Closing a project

Bulk close is the one operation that deliberately bypasses the step lifecycle:
it marks every open step `done`. That makes it able to mark unfinished work
finished, so it requires consent.

**From an agent (MCP), two calls:**

```
close_project(project_id)                          → plan, nothing changes
close_project(project_id, confirm: true,
               reason: "why it is closed")          → closes
```

The first call returns what would happen: the steps that would move to `done`
with their current statuses, and the unresolved blockers on them. The agent
shows that to a human; only after agreement does it call again. `reason` is
required on the confirming call — it is recorded as the project's closing
decision, so it is the only durable trace of why the project was closed.

**From the CLI, one command:**

```bash
pm project close 1 "shipped in v1.2"
```

No confirmation prompt: the person typing it is the consent, so there is
nothing to preview. `reason` is still required.

Two things the plan makes explicit:

- Closing does **not** resolve blockers. The records stay on the completed
  steps. A blocker on a closed step is visible in `list_blockers` and still
  carries its reason.
- The close is not atomic: steps are written one by one, so an interrupted
  close can leave some steps done. Re-running it finishes the job — steps
  already `done` are skipped, and the project is not yet `completed`.

### Known limit

An agent holding a valid token can send `confirm: true` on the first call. The
daemon cannot distinguish "the human agreed" from "the agent decided". This
flow makes the agent show the plan; it does not prove a human read it. A real
stop needs a confirmation only a human can issue.

## Trash and backups

Deleting a project moves it to `.trash`, never away. Both surfaces share it:

```bash
pm trash list                        # what is in the bin, with numbers and titles
pm trash restore <name|number|title> # bring one back; ambiguous matches fail with candidates
pm trash clean                       # erase forever. CLI only, never over MCP.
```

Agents get `trash_list` and `trash_restore` with the same rules. There is no
`trash_clean` tool on purpose: an agent may bin things, only a human may shred
them. The same split holds for close: MCP closes in two calls with consent,
CLI closes in one.

Every deletion - project, step, blocker, decision - also snapshots the exact
bytes into `_meta/backups/<unix>/`, keeping the newest 20 runs. The trash
covers projects; the backups cover everything else, because deleting a step
has no undo button. Recovery from a backup is manual: copy the files back
from the run directory (they sit under git, so `git log` shows what changed).

Restore refuses instead of merging: if a live project already holds the
number, or the target dir exists, the call fails with the reason and nothing
moves.

## Timestamps

Event times (`created_at`, `updated_at`, `completed_at`, decision `date`) are
RFC3339 in UTC, second precision:

```yaml
updated_at: "2026-09-29T19:57:55Z"
```

The legacy date-only form is still **read** as start-of-day UTC, so old stores
keep working:

```yaml
updated_at: "2026-09-27"        # read as 2026-09-27T00:00:00Z
```

Each record is rewritten in the canonical form the next time it is written, so
a store migrates itself as it is touched. `pm doctor` reports how far along it
is:

```
Метки времени: 12 устаревших (только дата), 0 битых
```

Legacy is not an error — the value is read correctly, it just carries no time
of day, so two events on the same day cannot be ordered. `doctor` does not
rewrite them: writing outside the daemon would reintroduce the races the daemon
exists to prevent.

A timestamp that cannot be parsed at all is kept verbatim, logged as a warning,
and reported in `data_warnings` in the briefing response — the counts are
partial, and the caller can see that they are. It never fails the whole file,
and it is never silently replaced with a zero time.

## Reading through MCP

Reads come in three sizes, so asking one question does not pull a whole project
into your context.

| Question | Tool | Cost |
|---|---|---|
| "Where does this stand?" | `get_project` | summary: counts by status, open blockers, last completed step, plus a hint naming the next call |
| "Which steps are there?" | `list_steps` | id, title, status, updated_at, blocker_ids per step |
| "What is on this one step?" | `get_step` | that step in full, with blockers and artifacts |
| "What blocks this step?" | `list_steps` → `blocker_ids`, then `get_step` or `list_blockers` with `step_id` | ids are cheap, reasons stay in the detail level |

`get_project` used to return everything, and a detailed read of one project
cost more than listing every project — 2 195 B against 1 755 B, which inverts
the expected ratio. It is now roughly 5× smaller. Pass `detail: true` for the
old full dump when you genuinely need every step and decision.

The summary counts every step (`steps_by_status` sums to `steps_total`), so
nothing is hidden — it is just not spelled out until you ask.

### Days and time zones

The briefing buckets events by calendar day in the **daemon's** time zone, not
UTC. A step closed at 01:00 MSK belongs to today, not to the previous UTC day.
Because both `pm briefing` and `get_briefing` are computed by the daemon, they
always agree on what "today" means.

## Reliability

| Property | Mechanism |
|----------|----------|
| Single writer | only `pm serve` touches YAML; clients go over HTTP |
| Atomic writes | temp → sync → rename |
| Durability | fsync file + parent dir |
| Cross-process locking | flock on `.pm.lock` |
| Auth | Bearer token (sops-managed file, `LoadCredential`, never in the nix store) |
| Graceful shutdown | signal.NotifyContext (CLI serve + MCP) / systemd restart=always |
| Recovery | trash (.trash), NextNumber fallback scan |
| Diagnostics | `pm doctor` checks integrity (host-local) |

## Tests

```bash
go test ./... -count=1
go vet ./...
```

## License

MIT
