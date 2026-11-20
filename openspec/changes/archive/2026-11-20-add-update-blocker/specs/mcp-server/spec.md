# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Blocker update tool

The server SHALL expose `update_blocker` with required `project_id`,
`step_id`, `blocker_id` and at least one of `title` or `reason`. It edits the
text of a stored blocker and MUST keep the blocker id unchanged, because
notes and links reference the id rather than the current wording. Status is
not editable here: resolving a blocker stays a separate operation.

#### Scenario: Reword a blocker

- **WHEN** `update_blocker` is called with an existing blocker and a new non-empty `title`
- **THEN** the stored blocker carries the new title with the same id and unchanged reason and status

#### Scenario: Correct the reason only

- **WHEN** `update_blocker` is called with `reason` and no `title`
- **THEN** only the reason changes and the title stays as it was

#### Scenario: No field to update

- **WHEN** `update_blocker` is called with neither `title` nor `reason`
- **THEN** the call fails naming the two accepted fields, and nothing is written

#### Scenario: Reword collides with another blocker

- **WHEN** the new title slugs to an id already used by a different blocker in the same step
- **THEN** the call fails naming the colliding id and both blockers keep their text

#### Scenario: Unknown blocker or step

- **WHEN** `update_blocker` is called with a `blocker_id` that is not in that step, or a `step_id` that is not in the project
- **THEN** the call fails with a not-found error quoting the missing reference
