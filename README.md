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
| `get_project` | project + steps + decisions (JSON) |
| `add_project` | create a project |
| `add_step` | add a step |
| `start_step` | todo → in_progress |
| `review_step` | send to review |
| `done_step` | finish (review only) |
| `add_blocker` | add a blocker |
| `resolve_blocker` | resolve a blocker |
| `add_decision` | record a decision |
| `get_briefing` | generate a digest |
| `list_steps` | project steps (JSON) |
| `list_blockers` | blockers (JSON) |
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
created_at: "2026-06-13"
updated_at: "2026-06-13"
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
    created_at: "2026-06-10"
    updated_at: "2026-06-10"
```

```yaml
# <project-id>/decisions/<slug>.yaml
id: use-go
title: "Go as language"
reason: "One binary, no dependencies"
date: "2026-06-13"
project_id: "..."
```

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
