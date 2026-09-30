# cli-lifecycle Specification (delta)

## ADDED Requirements

### Requirement: Bulk close stays a single CLI command

`pm project close <project-id> <reason>` SHALL close in one command, unchanged
by the consent requirement that applies to agent callers. The person typing it
is the consent; there is nothing to preview for a human who is already at the
terminal deciding. `reason` SHALL be required, since it is the only durable
trace of why the project was closed.

An empty or missing reason SHALL be refused rather than defaulted: the default
`bulk close` recorded by the store is acceptable for API callers that pass an
empty string, but the CLI has a human present and can require the real reason.

#### Scenario: Human closes with one command

- **WHEN** a user runs `pm project close 3 "shipped in v1.2"` on an active project
- **THEN** open steps become `done`, the project becomes `completed`, and a
  `Closed: shipped in v1.2` decision exists

#### Scenario: Reason is required at the terminal

- **WHEN** a user runs `pm project close 3` with no reason
- **THEN** the command refuses and the project stays `active`
