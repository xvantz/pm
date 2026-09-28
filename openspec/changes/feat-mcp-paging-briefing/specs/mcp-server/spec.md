# mcp-server Specification (delta)

## ADDED Requirements

### Requirement: Message IDs and bounded reads

The server SHALL accept string and integer JSON-RPC `id` values (no dropping string IDs). Read tools returning unbounded dumps SHALL offer bounded modes (limits/truncation); `get_project` on large projects MUST NOT blow up agent context by default.

#### Scenario: String ID round-trips

- **WHEN** the client sends `tools/call` with a string `id`
- **THEN** the response carries the same string `id`
