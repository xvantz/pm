# cli-lifecycle Specification (delta)

## ADDED Requirements

### Requirement: Force completion

`done_step` (MCP) and `pm step done --force` (CLI) SHALL allow direct transition `todo/in_progress -> done` when an explicit `reason` is supplied. Every force completion MUST record an auto-decision `force-completed` with the reason for audit. Without the flag, `done` from a non-`review` state MUST keep failing.

#### Scenario: Force close from in_progress

- **WHEN** the agent calls `done_step` with `force: true` and `reason: "trivial, verified by build"` on an `in_progress` step
- **THEN** the step becomes `done` and a `force-completed` decision with that reason appears in the project

#### Scenario: Default stays strict

- **WHEN** the agent calls `done_step` without `force` on an `in_progress` step
- **THEN** the call fails and the step stays `in_progress`
