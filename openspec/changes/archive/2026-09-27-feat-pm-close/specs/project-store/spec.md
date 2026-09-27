# project-store Specification (delta)

## ADDED Requirements

### Requirement: Bulk close contract

`Store` SHALL expose `CloseProject(projectID, reason string)`: every non-done step transitions to `done` with fresh `UpdatedAt`, the project becomes `completed` with `CompletedAt`, and a `Closed: <reason>` decision is recorded (empty reason defaults to `bulk close`). Closing an already completed project is an error. Lifecycle validation is intentionally bypassed here: archival must not cost N calls, and the reason decision keeps the audit trail.

#### Scenario: Bulk close open project

- **WHEN** `CloseProject` runs on a project with todo and in-progress steps
- **THEN** all steps are `done`, status is `completed`, and one `Closed:` decision exists

#### Scenario: Double close rejected

- **WHEN** `CloseProject` runs on a `completed` project
- **THEN** an `already completed` error is returned and nothing changes
