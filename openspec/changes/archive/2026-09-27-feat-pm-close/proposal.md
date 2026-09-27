# Proposal: feat-pm-close

## Problem

Closing a finished project costs N lifecycle calls (GCP took 21:
start → review → done × 7). There is no bulk close and no delete at all.
The strict lifecycle is right for active work (it caught real bugs),
but for archival it is pure friction.

Out of scope: moving Obsidian notes. The daemon has no vault access
by design; the agent moves `1. Projects/x.md` → `4. Archive/` in chat
(like post archiving). One call cannot span two systems honestly.

## Solution

- `CloseProject(id, reason)`: force-completes every non-done step,
  sets project `completed` + `CompletedAt`, records a `Closed: <reason>`
  decision. One call, reason preserved.
- `delete_project`: exposes the existing trash move as an MCP tool
  (CLI `del project` and API DELETE already exist).
- Tool descriptions rewritten to be searchable on first try.

## Scope

- Store interface + FileStore + MockStore: `CloseProject`.
- CLI `pm project close <id> [--reason]`.
- API `POST /api/projects/{ref}/close`.
- MCP `close_project` + `delete_project`.
- Client + apistore coverage, tests, README, spec.
