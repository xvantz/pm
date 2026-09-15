# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Trash tools

The server SHALL expose `trash_list` (trashed items with names, ids and deletion timestamps) and `trash_restore` (restore by name or id). Ambiguous names MUST return an error listing candidates instead of restoring the first match. Both tools use `inputSchema` with `"type": "object"`.

#### Scenario: Restore by unambiguous name

- **WHEN** the client calls `trash_restore` with a name matching exactly one trashed item
- **THEN** the item is restored to its project and reported

#### Scenario: Ambiguous restore is rejected

- **WHEN** the client calls `trash_restore` with a name matching several trashed items
- **THEN** the call fails with the candidate list and nothing is restored
