# mcp-server Specification (delta)

## MODIFIED Requirements

### Requirement: Tool input schemas

Every tool inputSchema SHALL be a JSON Schema object with `"type": "object"` explicitly set. Empty schemas MUST use `{"type": "object", "properties": {}}` - a bare `{}` is forbidden because strict MCP clients (Hermes pydantic validation of `ListToolsResult`) reject the entire `tools/list` response when any single schema lacks `type`.

#### Scenario: List tools passes strict validation

- **WHEN** an initialized client sends `tools/list`
- **THEN** every tool entry carries `inputSchema.type == "object"` and the response validates against the MCP `ListToolsResult` schema

#### Scenario: Call list_projects

- **WHEN** the client calls `list_projects` with `{}` arguments
- **THEN** the handler returns the project list as text content
