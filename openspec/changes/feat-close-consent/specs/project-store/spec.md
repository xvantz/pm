# project-store Specification (delta)

## MODIFIED Requirements

### Requirement: Bulk close contract

`Store` SHALL expose `CloseProject(projectID, reason string)`: every non-done step transitions to `done` with fresh `UpdatedAt`, the project becomes `completed` with `CompletedAt`, and a `Closed: <reason>` decision is recorded (empty reason defaults to `bulk close`). Closing an already completed project is an error. Lifecycle validation is intentionally bypassed here: archival must not cost N calls, and the reason decision keeps the audit trail.

`Store` SHALL also expose `ClosePlan(projectID) (*ClosePlan, error)`: a read-only
preview of what `CloseProject` would do — the steps that would move to `done`
with their current statuses, and the unresolved blockers they carry. It MUST
NOT mutate anything. `CloseProject` remains the mutating operation and keeps its
existing contract; the plan exists so a caller can show a human what an
irreversible bulk close is about to do before performing it.

#### Scenario: Bulk close open project

- **WHEN** `CloseProject` runs on a project with todo and in-progress steps
- **THEN** all steps are `done`, status is `completed`, and one `Closed:` decision exists

#### Scenario: Double close rejected

- **WHEN** `CloseProject` runs on a `completed` project
- **THEN** an `already completed` error is returned and nothing changes

#### Scenario: Plan describes the close without applying it

- **WHEN** `ClosePlan` runs on a project with open steps and an unresolved blocker
- **THEN** the plan lists those steps with their statuses and the blocker, and
  the stored project is unchanged
