# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Actionable responses

Every MCP tool success response that implies a next action SHALL name the next
tool by its exact catalogue name with its key parameters, never a CLI command:
the MCP agent has tools, not a shell with `pm`. Every tool error for a
recoverable caller mistake (unknown step id, duplicate id, missing argument,
unmet lifecycle precondition, consent without reason) SHALL state what
happened, what is needed instead, and which call fixes it.

Rationale: a hint in a language the caller cannot execute (`Next: pm step
start 1 x` to an agent with no shell) is zero signal at full byte cost. An
error without the fix (`step_id is required`) buys a second failed call where
one sentence would do. Sources: Anthropic effective-tools (actionable errors
with a correct-call example) and Merge.dev (workflow in text, ops in schema).

#### Scenario: Hints name tools

- **WHEN** any write tool succeeds with a next action
- **THEN** the response names the next tool (e.g. `start_step`,
  `list_blockers`) and carries no `pm <cli ...>` command text

#### Scenario: Step hint routes by state

- **WHEN** the client calls `get_step` on a blocked step
- **THEN** the response hints at `resolve_blocker`/`list_blockers` for the
  reason; on a plain step it hints at the next lifecycle call for its status

#### Scenario: Unknown step names the fix

- **WHEN** the client calls with a step id that does not exist
- **THEN** the error names the step and points at `list_steps` as the source
  of live ids

#### Scenario: Duplicate names the way out

- **WHEN** the client adds a step, blocker or decision whose id already exists
- **THEN** the error names the conflict and says to reuse or rename

#### Scenario: Unmet precondition names the repair call

- **WHEN** the client calls `done_step` on a non-review step
- **THEN** the error states review is required and names `review_step` as the
  fix; `close_project` confirm without reason names `reason` and the two-call
  flow
