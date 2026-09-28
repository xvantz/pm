# project-store Specification (delta)

## ADDED Requirements

### Requirement: Event timestamps

All `CreatedAt`/`UpdatedAt`/`Date` fields SHALL be RFC3339 timestamps. Readers MUST accept legacy `YYYY-MM-DD` values as start-of-day UTC. String comparison of timestamps MUST NOT be used for ordering; parse then compare.

#### Scenario: Legacy date reads as midnight

- **WHEN** a YAML file carries `UpdatedAt: "2026-09-27"`
- **THEN** it parses as start-of-day UTC and orders before any timed event that day
