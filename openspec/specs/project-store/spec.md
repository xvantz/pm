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

`Store` SHALL expose `CloseProject(projectID, reason string, confirm bool)
(*ClosePlan, error)`: every non-done step transitions to `done` with fresh
`UpdatedAt`, the project becomes `completed` with `CompletedAt`, and a
`Closed: <reason>` decision is recorded. Closing an already completed project is
an error. Lifecycle validation is intentionally bypassed here: archival must not
cost N calls, and the reason decision keeps the audit trail.

`confirm` gates the irreversible half and belongs in the store contract, not only
in a caller: with `confirm=false` the call returns the plan and mutates nothing,
with `confirm=true` it requires a non-empty `reason` and closes. A non-empty
reason is mandatory — there is no default, because the decision is the only
durable trace of why the project was closed.

Closing does NOT resolve blockers: the blocker records stay on the completed
steps.

`Store` SHALL also expose `ClosePlan(projectID) (*ClosePlan, error)`: a read-only
preview of what `CloseProject` would do — the steps that would move to `done`
with their current statuses, and the unresolved blockers on them. It MUST NOT
mutate anything. On a `completed` project it returns an empty plan, not an
error. The plan exists so a caller can show a human what an irreversible bulk
close is about to do before performing it.

#### Scenario: Bulk close open project

- **WHEN** `CloseProject` runs with `confirm: true` and a reason on a project with todo and in-progress steps
- **THEN** all steps are `done`, status is `completed`, and one `Closed:` decision exists

#### Scenario: Double close rejected

- **WHEN** `CloseProject` runs on a `completed` project
- **THEN** an `already completed` error is returned and nothing changes

#### Scenario: Preview mutates nothing

- **WHEN** `CloseProject` runs with `confirm: false`
- **THEN** it returns a plan, and the project's status, steps and decisions are unchanged

#### Scenario: Confirm without a reason is refused

- **WHEN** `CloseProject` runs with `confirm: true` and an empty or blank reason
- **THEN** it returns an error naming the reason, and nothing changed

#### Scenario: Plan describes the close without applying it

- **WHEN** `ClosePlan` runs on a project with open steps and an unresolved blocker
- **THEN** the plan lists those steps with their statuses and the blocker, and
  the stored project is unchanged

### Requirement: Event timestamps

`CreatedAt`, `UpdatedAt`, `Date` and `CompletedAt` SHALL be typed as
`types.Timestamp` (a `time.Time` wrapper), never as bare `string`. The type
SHALL expose the only sanctioned way to read or compare an event time, so
string comparison of timestamps is impossible by construction.

Writes SHALL be RFC3339 in UTC with a `Z` suffix, second precision, no
fractional part (`2026-09-29T19:57:55Z`). Offsets MUST NOT be written.

Readers MUST accept the legacy `YYYY-MM-DD` form and MUST interpret it as
start-of-day UTC.

`types.NowTimestamp` SHALL return the canonical form. There SHALL be no
string-returning variant of the current-time helper: it would be an escape
hatch back to comparing event times as text.

Legacy data migrates on write: a record read in the legacy form is rewritten in
canonical form the next time it is touched, so a store converts itself as it is
used. No separate migration pass exists — it would do exactly what an ordinary
write does, only outside the daemon. `doctor` SHALL report the count of
remaining legacy and unreadable records and SHALL NOT rewrite them, because
bypassing the daemon to write reintroduces the races the daemon removed.

#### Scenario: New write carries a timestamp

- **WHEN** a step transitions to `done`
- **THEN** its `updated_at` parses as RFC3339 UTC with a `Z` suffix

#### Scenario: Legacy date reads as midnight

- **WHEN** a YAML file carries `updated_at: "2026-09-27"`
- **THEN** it reads as start-of-day UTC and orders before any timed event on
  that calendar day

#### Scenario: Empty field stays absent

- **WHEN** an entity with no `completed_at` is written
- **THEN** the field is omitted entirely, not written as a zero time

#### Scenario: Type forbids string comparison

- **WHEN** code compares an event time to a `string` literal
- **THEN** compilation fails; no runtime string comparison of timestamps exists

#### Scenario: Doctor reports legacy records

- **WHEN** `doctor` runs against a store holding `updated_at: "2026-09-27"`
- **THEN** it reports the legacy record count and does not modify the files

#### Scenario: Legacy record rewrites itself on the next write

- **WHEN** a project read from a legacy `updated_at: "2026-09-27"` is saved again
- **THEN** the stored value is `2026-09-27T00:00:00Z` and no other field changes

### Requirement: Calendar-day bucketing

Consumers that group events by day (the briefing is the only one today) SHALL
compare event times as instants and bucket them by calendar day in the **time
zone of the process performing the read**, never by UTC day. A single
sanctioned helper SHALL be used by every surface (CLI and MCP) so that a
terminal run and an MCP call on the same store report identical counts.

Rationale: the single-writer daemon defines the day for all clients, so an
event closed at 01:00 local time belongs to the current day, not to the
previous UTC day.

#### Scenario: Local-time event lands in the local day

- **WHEN** a step is completed at `2026-09-29T01:00:00+03:00`
  (`2026-09-28T22:00:00Z`) and the reader runs in `Europe/Moscow`
- **THEN** the step counts toward that calendar day, not toward the previous one

#### Scenario: Surfaces agree

- **WHEN** `pm briefing` and `get_briefing` read the same store in the same zone
- **THEN** their step and project counts are identical

### Requirement: Unreadable timestamps are loud

Any event time that cannot be parsed SHALL NOT be silently replaced with a
zero time. The consuming code SHALL emit a `slog.Warn` carrying the entity id,
the field name and the raw value, and the briefing response SHALL carry an
entry in `data_warnings`. A single unreadable timestamp SHALL NOT fail the whole
briefing, but the caller MUST be able to observe that the numbers are partial.

Silent substitution of a zero time is prohibited: an agent reading a briefing
MUST be able to tell complete counts from degraded ones.

#### Scenario: Unparseable timestamp degrades loudly

- **WHEN** a step carries `updated_at: "2026-13-45"`
- **THEN** a warn log names the project, step, field and raw value, the response
  `data_warnings` is non-empty, and the briefing still returns

#### Scenario: Clean store has no warnings

- **WHEN** every event time in the store parses
- **THEN** `data_warnings` is empty

### Requirement: Delete backups

`FileStore` SHALL write a backup copy before every deletion
(`DeleteProject`, `DeleteStep`, `DeleteBlocker`, `DeleteDecision`) under
`_meta/backups/<unix-timestamp>/`, keeping only the 20 most recent backup
runs. The backup SHALL contain the exact bytes being removed, so a hard
delete of a step, blocker or decision - which has no trash - stays manually
recoverable from files under git.

Backups are written on delete only, never on save. Nothing reads them back
automatically; recovery is a human copying files, documented in the README.

#### Scenario: Step delete leaves a backup

- **WHEN** a step is deleted
- **THEN** its YAML bytes exist under the newest `_meta/backups/` run

#### Scenario: Rotation keeps twenty runs

- **WHEN** 21 deletions happen in a row
- **THEN** exactly 20 backup run directories remain
