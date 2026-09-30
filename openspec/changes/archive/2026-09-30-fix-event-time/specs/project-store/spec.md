# project-store Specification (delta)

## ADDED Requirements

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
