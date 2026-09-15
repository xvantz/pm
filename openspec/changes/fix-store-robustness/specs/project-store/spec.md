# project-store Specification (delta)

## MODIFIED Requirements

### Requirement: Project resolution

`ResolveProject` SHALL resolve numbers, full UUIDs and prefixes in a single directory scan per call. Corrupt project directories MUST surface as `unreadable` entries (directory name + error) instead of being silently skipped, so no data loss goes unnoticed.

#### Scenario: Resolve by number

- **WHEN** the caller passes `"1"` and a project with number 1 exists
- **THEN** its full project data is returned

#### Scenario: Ambiguous prefix

- **WHEN** the caller passes a prefix matching more than one project id
- **THEN** an `ambiguous project prefix` error is returned

#### Scenario: Single-scan resolve

- **WHEN** the caller resolves any ref on a store with 50 projects
- **THEN** at most one directory scan occurs (benchmark-guarded)

#### Scenario: Corrupt project is visible

- **WHEN** a project directory contains unparsable YAML
- **THEN** `ListProjects` includes it as `unreadable` with the parse error, and `doctor` reports it with repair hints

### Requirement: Monotonic project numbers

`next_number` increments SHALL be atomic with project creation under the same lock: N concurrent `add_project` calls MUST yield N distinct numbers.

#### Scenario: Sequential project creation

- **WHEN** two projects are created one after another
- **THEN** the second project number is greater than the first
