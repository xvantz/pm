# mcp-server Specification (delta)

## MODIFIED Requirements

### Requirement: Tool catalogue

The server SHALL expose the tool set registered in `RegisterPMTools` (`internal/mcp/tools.go`): the live `tools/list` response IS the catalogue, and no tool count is hardcoded in this spec. Every entry SHALL carry `name`, `description` and `inputSchema`. Integrity invariants (non-empty set, unique names, every `inputSchema.type == "object"`) are asserted by `TestToolsList_AllSchemasAreObjects`, which runs against all registered tools: newly registered tools are covered without spec edits.

#### Scenario: List tools

- **WHEN** an initialized client sends `tools/list`
- **THEN** the server returns all registered tools with `name`, `description` and `inputSchema`

#### Scenario: New tool needs no catalogue edit

- **WHEN** a tool is added to `RegisterPMTools` with a valid schema and unique name
- **THEN** `tools/list` includes it and all spec validations still pass unchanged
