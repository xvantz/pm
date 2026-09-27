# project-store Specification (delta)

## MODIFIED Requirements

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
