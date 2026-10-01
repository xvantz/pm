# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Trash tools

The server SHALL expose `trash_list` (trashed items with trash name, project
id, number, title and deletion time) and `trash_restore` (restore by trash
name, project number or title). Ambiguous matches MUST return an error listing
candidates instead of restoring anything. Unknown names MUST fail naming the
requested value. Both tools use `inputSchema` with `"type": "object"`.

#### Scenario: Restore by unambiguous name

- **WHEN** the client calls `trash_restore` with a name matching exactly one trashed item
- **THEN** the item is restored to its project and reported

#### Scenario: Ambiguous restore is rejected

- **WHEN** the client calls `trash_restore` with a name matching several trashed items
- **THEN** the call fails with the candidate list and nothing is restored

#### Scenario: Unknown restore target fails

- **WHEN** the client calls `trash_restore` with a name matching nothing
- **THEN** the call fails naming the requested value

### Requirement: No permanent wipe over MCP

The server SHALL NOT expose `trash_clean` or any permanent-delete tool over
MCP. `delete_project` moves to trash; recovery is via `trash_restore`. Only
the CLI may permanently erase (`pm del`, `trash clean`).

#### Scenario: Catalogue has no wipe tool

- **WHEN** the client lists tools
- **THEN** no tool name implies permanent deletion of trashed items
