# serve-api Specification (delta)

## ADDED Requirements

### Requirement: Close confirm carries the plan

`POST /api/projects/{ref}/close` with `confirm: true` SHALL return both the
closed project and the plan built before closing (`{confirmed: true, project,
plan}`), not the project alone. The plan lets every caller report how many
steps moved without a second read, and keeps file-backed and daemon-backed
stores observably identical: both answer confirm with a plan.

Rationale: the daemon answered confirm with `plan: nil` while `MockStore`
answered with a plan, so MCP crashed only in production (nil dereference in
`handleCloseProject`, SIGSEGV killing pm-mcp, found 2026-10-03). A contract
both sides share removes the divergence instead of guarding one caller.

#### Scenario: Confirm answers with both

- **WHEN** the client posts `close` with `confirm: true` and a reason
- **THEN** the response carries the closed project and the non-null plan,
  and the plan's step count matches the steps moved to done
