# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Reads are sized to the question

`get_project` SHALL return a project summary by default and SHALL NOT return
the step list unless asked. The summary SHALL carry: number, title, status,
goal, tags, step counts grouped by status, the count of unresolved blockers,
and the most recently completed step (title and when). It SHALL NOT carry the
full step objects or the decision list.

Called with `detail: true`, it SHALL return the full project data as before.

`list_steps` SHALL return per step only `id`, `title`, `status` and
`updated_at`; it SHALL NOT embed blockers or artifacts.

Rationale: measured, one detailed project cost more than the list of all
projects (2 195 B vs 1 755 B), which inverts the expected ratio — a detail
read must cost more than an overview, not less. Most calls want "where does
this stand", which the summary answers in a fraction of the bytes.

#### Scenario: Default read is a summary

- **WHEN** the client calls `get_project` with only `project_id`
- **THEN** the response carries the project fields, step counts by status and
  the unresolved blocker count, and carries no step objects

#### Scenario: Detail flag restores the full dump

- **WHEN** the client calls `get_project` with `detail: true`
- **THEN** the response carries the full project data including every step

#### Scenario: Summary is markedly smaller

- **WHEN** a project is read both ways
- **THEN** the summary response is substantially smaller than the detailed one

#### Scenario: Step listing omits heavy fields

- **WHEN** the client calls `list_steps`
- **THEN** each step carries id, title, status and updated_at, and no blockers
  or artifacts

### Requirement: Single-step reads

The server SHALL expose a `get_step` tool taking `project_id` and `step_id`,
returning that step in full: its fields, its blockers, its artifacts and its
timestamps.

Rationale: without it, answering "what is on this one step" costs the whole
project, which makes the two-level split above useless for large projects —
the detail level would still return 20 steps when the caller wanted one.

#### Scenario: One step in full

- **WHEN** the client calls `get_step` for an existing step
- **THEN** the response carries that step with its blockers and artifacts

#### Scenario: Missing step is an error

- **WHEN** the client calls `get_step` for a step that does not exist
- **THEN** the call fails naming the step and the project

#### Scenario: Unknown step in a real project

- **WHEN** the client calls `get_step` with a step id from another project
- **THEN** the call fails naming the step; it does not fall back to a list
