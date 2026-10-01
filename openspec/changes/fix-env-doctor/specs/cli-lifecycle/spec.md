# cli-lifecycle Specification (delta)

## MODIFIED Requirements

### Requirement: Doctor

`pm doctor` SHALL ask the daemon for its integrity verdict (`GET /api/doctor`)
and print it: counts, orphans, unreadable files and timestamp stats, plus
actionable repair hints. There SHALL be no local file scan: the daemon is the
single reader, so there is nothing to compare and nothing to diverge.

When the daemon is unreachable, doctor SHALL fail with an error naming the
daemon, not fall back to a local scan. A fallback would reintroduce the second
reader this change removes. Fix the daemon.

#### Scenario: Missing store

- **WHEN** the user runs `pm doctor` with no storage directory present
- **THEN** the daemon reports the store as missing and doctor suggests running `pm init`

#### Scenario: Daemon down is the verdict

- **WHEN** the daemon is unreachable
- **THEN** doctor fails naming the daemon and performs no local file scan
