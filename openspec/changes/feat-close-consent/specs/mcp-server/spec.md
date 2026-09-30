# mcp-server Specification (delta)

## MODIFIED Requirements

### Requirement: Bulk close tool

The `close_project` tool SHALL require explicit consent from the caller, in two
calls. Called without `confirm`, it SHALL NOT close: it returns a plan — the
steps that would move to `done` with their current statuses, and the unresolved
blockers on those steps — and the project is untouched. Called with
`confirm: true`, it SHALL require a non-empty `reason` and then bulk-close:
force-complete every non-done step, set status `completed` with `CompletedAt`,
and record a `Closed: <reason>` decision. Input schema is `{"type": "object"}`
with required `project_id` and optional `reason` and `confirm`.

Closing does NOT resolve blockers: the blocker records stay on the completed
steps. The plan states this, so a human is not left believing a blocker was
settled when only the step around it was.

Rationale: the tool deliberately bypasses the step lifecycle, so it is the one
operation that can mark unfinished work finished. Requiring a plan first is what
keeps the `review` stop meaningful — the confirmation is the human's claim that
the work is done, relayed by the agent, not the agent's own judgement. For
finished work it still replaces N lifecycle calls; for active work the strict
lifecycle stays.

#### Scenario: Close with reason

- **WHEN** the client calls `close_project` with a valid `project_id` and `reason`
- **THEN** all open steps become `done`, status is `completed`, and a `Closed:` decision carries the reason

#### Scenario: Close already completed

- **WHEN** the client closes a project whose status is already `completed`
- **THEN** the call fails with an `already completed` error

#### Scenario: First call plans instead of closing

- **WHEN** the client calls `close_project` without `confirm` on a project with
  open steps and an unresolved blocker
- **THEN** the project stays `active`, steps keep their statuses, and the
  response lists the steps, the blocker, and that the blocker stays on the record

#### Scenario: Confirm without a reason is refused

- **WHEN** the client calls `close_project` with `confirm: true` and an empty reason
- **THEN** the call fails, the project stays `active`, and no decision is written
