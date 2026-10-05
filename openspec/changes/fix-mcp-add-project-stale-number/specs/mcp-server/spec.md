# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Project creation echoes the server-assigned number

The `add_project` tool SHALL print the server-assigned project number,
re-read by stable UUID after save: the advisory `NextNumber` known before
save may be stale whenever the daemon counter ran ahead of the live max
(trashed projects keep their numbers consumed, concurrent creates race),
and the follow-up hint MUST point at the real project. The `Next:` hint
SHALL carry the client-assigned UUID (`add_step {project_id: "<uuid>"}`),
which is stable across every backend; the `Project #N` headline SHALL carry
the re-read number. A failed re-read SHALL be a loud error, never a
confirmation with the advisory number.

Rationale: the remote daemon assigns numbers from a file counter that counts
trashed projects, while the advisory number is live max+1. Printing the
advisory value lies permanently after any delete (observed 2026-10-05:
printed #10, real #14, `add_step` 404, gateway circuit breaker parked the
server). The CLI already re-reads (`cli-lifecycle / Creation echoes the
server-assigned number`); this requirement closes the MCP half of the same
contract.

#### Scenario: Confirmation survives trash gaps

- **WHEN** the client creates a project while the daemon counter sits ahead
  of the live max (numbers consumed by trash)
- **THEN** the response prints the daemon-assigned number and an `add_step`
  call with the printed `project_id` succeeds on the first try with no
  manual `list_projects`

#### Scenario: Failed re-read is loud

- **WHEN** the project cannot be read back by its UUID right after a
  successful save
- **THEN** the tool returns a `resolve created project` error instead of a
  confirmation carrying the advisory number
