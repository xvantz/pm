# Project Store Specification

## Purpose

`FileStore` (`internal/store/filestore.go`) persists projects, steps, blockers and decisions as YAML files under a root directory (e.g. `./pm/projects`), coordinated between concurrent CLI and MCP processes. Shared contract in `internal/store/store.go`, `internal/types/types.go`.

## Requirements

### Requirement: Atomic durable writes

All write operations SHALL be atomic and durable: write to a temp file, fsync to disk, then rename into place. Concurrent writers coordinate via POSIX flock on the project directory.

#### Scenario: Concurrent CLI and MCP writes

- **WHEN** `pm` CLI and `pm-mcp` write to the same project concurrently
- **THEN** no partial or torn YAML files appear; one write wins cleanly per file

### Requirement: Project resolution

`ResolveProject` SHALL accept a project number (e.g. `"1"`), a full UUID, or a UUID prefix. Ambiguous prefixes return an error naming the match count; unknown refs return a not-found error. Multi-scan resolution is acceptable at current scale; a single-scan index is deferred until a 50-project bench proves a win.

#### Scenario: Resolve by number

- **WHEN** the caller passes `"1"` and a project with number 1 exists
- **THEN** its full project data is returned

#### Scenario: Ambiguous prefix

- **WHEN** the caller passes a prefix matching more than one project id
- **THEN** an `ambiguous project prefix` error is returned

### Requirement: Listing tolerates corruption

`ListProjects` SHALL skip unreadable project directories with a `slog.Warn` log entry and return the rest, never failing the whole call. `doctor` surfaces orphans and parse errors with repair hints. Full `unreadable` records in list output are deferred to the trash-restore follow-up.

#### Scenario: One corrupt project does not break the list

- **WHEN** one project directory contains unparsable YAML and the client calls `list_projects`
- **THEN** the response contains all readable projects and the call succeeds

### Requirement: Monotonic project numbers

Project numbers SHALL be monotonic. Under the single-writer daemon, increments are serialized by the daemon mutex (E2E: 20 parallel creates, zero duplicates). The `_meta/next_number` counter file with directory-scan fallback remains for direct file access. A flock-atomic increment is superseded while all writes go through the daemon.

#### Scenario: Sequential project creation

- **WHEN** two projects are created one after another
- **THEN** the second project number is greater than the first

#### Scenario: Concurrent creates via daemon

- **WHEN** 20 `add_project` calls race through the daemon
- **THEN** all 20 yield distinct numbers

### Requirement: Bulk close contract

`Store` SHALL expose `CloseProject(projectID, reason string)`: every non-done step transitions to `done` with fresh `UpdatedAt`, the project becomes `completed` with `CompletedAt`, and a `Closed: <reason>` decision is recorded (empty reason defaults to `bulk close`). Closing an already completed project is an error. Lifecycle validation is intentionally bypassed here: archival must not cost N calls, and the reason decision keeps the audit trail.

#### Scenario: Bulk close open project

- **WHEN** `CloseProject` runs on a project with todo and in-progress steps
- **THEN** all steps are `done`, status is `completed`, and one `Closed:` decision exists

#### Scenario: Double close rejected

- **WHEN** `CloseProject` runs on a `completed` project
- **THEN** an `already completed` error is returned and nothing changes
