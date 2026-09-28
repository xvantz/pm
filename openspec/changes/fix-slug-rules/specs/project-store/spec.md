# project-store Specification (delta)

## ADDED Requirements

### Requirement: Slug contract

Step, blocker and decision IDs SHALL be lowercase dash-delimited slugs with no leading dots and no empty result. Deriving code lives only in `internal/slug`; api, cli and mcp MUST NOT reimplement it. Titles collapsing to the same slug within one scope conflict with `409`.

#### Scenario: Dotted title is filesystem-safe

- **WHEN** a step titled `...` is created
- **THEN** the request fails instead of creating a hidden file
