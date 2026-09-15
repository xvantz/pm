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

`ResolveProject` SHALL accept a project number (e.g. `"1"`), a full UUID, or a UUID prefix. Ambiguous prefixes return an error naming the match count; unknown refs return a not-found error.

#### Scenario: Resolve by number

- **WHEN** the caller passes `"1"` and a project with number 1 exists
- **THEN** its full project data is returned

#### Scenario: Ambiguous prefix

- **WHEN** the caller passes a prefix matching more than one project id
- **THEN** an `ambiguous project prefix` error is returned

### Requirement: Listing tolerates corruption

`ListProjects` SHALL skip unreadable project directories with a `slog.Warn` log entry and return the rest, never failing the whole call.

#### Scenario: One corrupt project does not break the list

- **WHEN** one project directory contains unparsable YAML and the client calls `list_projects`
- **THEN** the response contains all readable projects and the call succeeds

### Requirement: Monotonic project numbers

Project numbers SHALL be monotonic via the `_meta/next_number` counter file, with a full directory scan as first-run fallback for backward compatibility.

#### Scenario: Sequential project creation

- **WHEN** two projects are created one after another
- **THEN** the second project number is greater than the first
