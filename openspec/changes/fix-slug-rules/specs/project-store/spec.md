# project-store Specification (delta)

## ADDED Requirements

### Requirement: Slug contract

Step, blocker and decision IDs SHALL be derived by `slug.Of` and obey its
contract: lowercase, dash-delimited, only Unicode letters, Unicode numbers and
dashes. Every other character is either mapped to a dash (separators) or
dropped. No ID SHALL contain `.`, so no ID can hide a file; no ID SHALL
contain shell-special characters (`! ? #` and the like), because IDs are
filenames.

IDs SHALL be at most 200 bytes long (bytes, not runes: the filesystem counts
bytes and Cyrillic is multibyte). Longer titles are cut at a UTF-8 boundary
with any trailing dash trimmed, leaving headroom under NAME_MAX 255 with the
`.yaml` suffix. Titles cleaning down to nothing SHALL be rejected as invalid
before any write.

Two different titles MAY yield one slug. That SHALL be a conflict error
(`409` over HTTP, explicit failure in CLI and MCP before writing), never a
merge. Existing stored IDs are never rewritten by this rule.

#### Scenario: Specials are cleaned

- **WHEN** a step titled `Fix it! #urgent?` is created
- **THEN** its ID carries no `!`, `?` or `#`

#### Scenario: Dots cannot hide files

- **WHEN** a step titled `...` is created
- **THEN** the request fails instead of creating a hidden file

#### Scenario: Long titles are capped, not crashed

- **WHEN** a step titled with 300 characters is created
- **THEN** its ID is at most 200 bytes long and the step is stored

#### Scenario: Collision is a conflict

- **WHEN** `Hello World` exists and `hello_world` is created in the same project
- **THEN** the second create fails with a conflict and the first step is untouched
