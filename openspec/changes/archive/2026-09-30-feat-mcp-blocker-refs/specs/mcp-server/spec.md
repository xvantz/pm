# mcp-server Specification (delta)

## MODIFIED Requirements

### Requirement: Reads are sized to the question

`get_project` SHALL return a project summary by default and SHALL NOT return
the step list unless asked. The summary SHALL carry: number, title, status,
goal, tags, step counts grouped by status, the count of unresolved blockers,
and the most recently completed step (title and when). It SHALL NOT carry the
full step objects or the decision list.

Called with `detail: true`, it SHALL return the full project data as before.

`list_steps` SHALL return per step `id`, `title`, `status`, `updated_at` and
`blocker_ids` (ids of blockers on that step, empty when none); it SHALL NOT
embed blocker objects, reasons, or artifacts.

`list_blockers` SHALL accept an optional `step_id`. Without it the response
carries every step group as before. With it the response carries only the
group for that step; an unknown step SHALL be an error naming the step.

Rationale: measured, one detailed project cost more than the list of all
projects (2 195 B vs 1 755 B), which inverts the expected ratio — a detail
read must cost more than an overview, not less. Most calls want "where does
this stand", which the summary answers in a fraction of the bytes.

The `blocker_ids` list keeps the brief listing navigable: a `blocked` step
names its blockers without carrying their reasons, so the caller knows which
`get_step` or `resolve_blocker` to call next. A separate `get_blocker` tool
is rejected on purpose: blockers live inside steps (usually 0-2 per step)
and `get_step` already returns them with reasons, so a new tool would grow
the catalogue to save ~200 B.

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
- **THEN** each step carries id, title, status, updated_at and blocker_ids, and no blockers
  or artifacts

#### Scenario: Step listing names its blockers

- **WHEN** the client calls `list_steps` on a project with a blocked step
- **THEN** that step carries the blocker id in `blocker_ids`, and a step with no
  blockers carries an empty list

#### Scenario: Blocker listing filters by step

- **WHEN** the client calls `list_blockers` with `project_id` and `step_id`
- **THEN** the response carries only that step group; without `step_id` it carries
  every group as before
