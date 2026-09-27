# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Bulk close tool

The `close_project` tool SHALL bulk-close a finished project in one call: force-complete every non-done step, set status `completed` with `CompletedAt`, and record a `Closed: <reason>` decision. Input schema is `{"type": "object"}` with required `project_id` and optional `reason`. For finished work it replaces N lifecycle calls; for active work the strict lifecycle stays.

#### Scenario: Close with reason

- **WHEN** the client calls `close_project` with a valid `project_id` and `reason`
- **THEN** all open steps become `done`, status is `completed`, and a `Closed:` decision carries the reason

#### Scenario: Close already completed

- **WHEN** the client closes a project whose status is already `completed`
- **THEN** the call fails with an `already completed` error

### Requirement: Delete tool

The `delete_project` tool SHALL expose the existing trash move over MCP: the project leaves the list and lands in `.trash`, recoverable via `trash restore`. For finished work `close_project` is preferred over delete.

#### Scenario: Delete moves to trash

- **WHEN** the client calls `delete_project` with a valid `project_id`
- **THEN** the project disappears from the list and appears in trash
