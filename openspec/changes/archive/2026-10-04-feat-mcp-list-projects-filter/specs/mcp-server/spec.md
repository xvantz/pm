# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Project list filtering

The `list_projects` tool SHALL accept an optional `status` parameter with
values `active`, `completed` or `all`, defaulting to `active` when absent or
empty. The system SHALL filter in the MCP handler (tool layer); the store
contract and the daemon HTTP API SHALL stay unfiltered.

The response SHALL always carry `projects`, `count` (number of returned
entries), `active_count`, `completed_count` and `total`, so the cheap filtered
answer still COUNTs what it hid. When the default filtered view hides
completed projects (`completed_count > 0`), the response SHALL carry a hint
naming `status=all`.

An unknown `status` value SHALL be a loud error naming the accepted values,
never a silent default.

Rationale: measured on the live store, one third of the unfiltered list is
closed-project noise (3 of 9), growing linearly with every close. Sorting
reorders the bytes but does not remove them; only server-side filtering cuts
agent context and the `get_project` fan-out behind it. The counters keep the
honesty invariant from the sizing rule: filtered-out records are counted,
never silently dropped. `status=all` is the full-fidelity escape hatch, the
same shape as `detail:true` on `get_project`.

#### Scenario: Default lists active only but counts all

- **WHEN** the client calls `list_projects` with `{}` on a store with active and completed projects
- **THEN** `projects` carries only active entries, and `active_count + completed_count == total` with `count == len(projects)`

#### Scenario: Explicit all restores the full list

- **WHEN** the client calls `list_projects` with `status=all`
- **THEN** `projects` carries active and completed entries as before the change

#### Scenario: Completed-only view

- **WHEN** the client calls `list_projects` with `status=completed`
- **THEN** `projects` carries only completed entries with correct counts

#### Scenario: Unknown status is a loud error

- **WHEN** the client calls `list_projects` with a `status` outside `active|completed|all`
- **THEN** the call fails naming the accepted values instead of returning a silent default
