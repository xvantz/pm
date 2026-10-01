# cli-lifecycle Specification (delta)

## MODIFIED Requirements

### Requirement: Trash

Deleted projects SHALL go to trash: `trash list` shows trashed items with
their titles and deletion time, `trash restore <name>` brings them back by
trash name, project number or title. The same trash is exposed over MCP via
`trash_list` and `trash_restore`; permanent erase (`trash clean`, `pm del`)
stays CLI-only and is never exposed over MCP.

#### Scenario: Delete and restore

- **WHEN** the user deletes a project and runs `trash restore` with its name, number or title
- **THEN** the project reappears in the project list with its steps and decisions intact

#### Scenario: Ambiguous restore lists candidates

- **WHEN** the restore argument matches several trashed items
- **THEN** the call fails with the candidate list and nothing is restored
