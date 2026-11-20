# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Project update tool

The server SHALL expose `update_project` with a required `project_id` (number
or UUID, same as the read tools) and four optional fields: `title`, `goal`,
`tags`, `status`. Only fields present in the arguments are written; absent
fields keep their stored value and `tags: []` clears the list. A call carrying
none of the four fields MUST fail. The result SHALL report the stored project
after the write.

#### Scenario: Rename a project

- **WHEN** `update_project` is called with `project_id` and a non-empty `title`
- **THEN** the stored project carries the new title, keeps goal, tags and status unchanged, and the result shows the new title with the same project number

#### Scenario: Move a project out of the idea status

- **WHEN** `update_project` is called with `project_id` and `status: "active"`
- **THEN** the stored project has status `active` with `updated_at` advanced, and the result reports `active`

#### Scenario: Several fields in one call

- **WHEN** `update_project` is called with `title`, `goal` and `tags`
- **THEN** all three are stored in one write and the result reflects all three

#### Scenario: No mutable field is refused

- **WHEN** `update_project` is called with only `project_id`
- **THEN** the call fails with an argument error listing the accepted fields, and nothing is written

### Requirement: Project update validation

The call MUST reject an empty `title`, a `status` outside the domain set
(`idea`, `active`, `paused`, `completed`), and a `project_id` matching no
project. Every check runs before any write, so a rejected call leaves the
stored project untouched.

#### Scenario: Empty title is refused

- **WHEN** `update_project` is called with `title: ""`
- **THEN** the call fails with an argument error naming the title, and the stored project is unchanged

#### Scenario: Unknown project

- **WHEN** `update_project` is called with a reference that matches no project
- **THEN** the call fails with a not-found error quoting the reference, and no file is written

#### Scenario: Bad status value

- **WHEN** `update_project` is called with `status: "archived"`
- **THEN** the call fails with an argument error naming the accepted statuses, and the stored status is unchanged
