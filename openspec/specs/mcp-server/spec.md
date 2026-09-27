# MCP Server Specification

## Purpose

`pm-mcp` exposes Project Memory operations as MCP tools over stdio transport so LLM agents can read, create and update project data. Source: `cmd/pm-mcp/main.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`.

## Requirements

### Requirement: Stdio JSON-RPC transport

The server SHALL communicate over stdio using newline-delimited JSON-RPC 2.0 (NDJSON, one JSON object per line, no `Content-Length` framing) on both read and write, matching the Hermes Python MCP client. If the official MCP Go SDK is adopted, its stdio transport SHALL be used instead of any hand-rolled framing.

#### Scenario: Initialize handshake

- **WHEN** the client sends `initialize` with any params
- **THEN** the server responds with `protocolVersion: "2024-11-05"`, empty `capabilities.tools` and `serverInfo` (name, version), without requiring prior state

#### Scenario: Initialize handshake (Hermes interop)

- **WHEN** Hermes sends `initialize` as a single `\n`-terminated JSON line followed by `notifications/initialized`
- **THEN** the server responds with `protocolVersion`, `capabilities` and `serverInfo` as single `\n`-terminated JSON lines, with no `Content-Length` headers on the wire

#### Scenario: Calls before initialization are rejected

- **WHEN** the client sends `tools/list` or `tools/call` before `notifications/initialized`
- **THEN** the server responds with error code `-32000` ("Not initialized")

#### Scenario: Unknown methods

- **WHEN** the client sends an unknown method with an id
- **THEN** the server responds with error `-32601`; notifications without id are silently ignored

### Requirement: Tool catalogue

The server SHALL expose exactly 14 tools: `list_projects`, `get_project`, `add_project`, `add_step`, `start_step`, `review_step`, `done_step`, `add_blocker`, `resolve_blocker`, `add_decision`, `get_briefing`, `list_steps`, `list_blockers`, `list_decisions`.

#### Scenario: List tools

- **WHEN** an initialized client sends `tools/list`
- **THEN** the server returns all registered tools with `name`, `description` and `inputSchema`

### Requirement: Tool input schemas

Every tool inputSchema SHALL be a JSON Schema object with `"type": "object"` explicitly set. Empty schemas MUST use `{"type": "object", "properties": {}}` - a bare `{}` is forbidden because strict MCP clients (Hermes pydantic validation of `ListToolsResult`) reject the entire `tools/list` response when any single schema lacks `type`.

#### Scenario: List tools passes strict validation

- **WHEN** an initialized client sends `tools/list`
- **THEN** every tool entry carries `inputSchema.type == "object"` and the response validates against the MCP `ListToolsResult` schema

#### Scenario: Call list_projects

- **WHEN** the client calls `list_projects` with `{}` arguments
- **THEN** the handler returns the project list as text content

### Requirement: Tool call envelope

`tools/call` SHALL dispatch by `name`, pass `arguments` to the handler, and wrap the handler's string result as `content: [{type: "text", text}]`. Unknown tool names SHALL return `-32602`, handler failures SHALL return `-32603`.

#### Scenario: Unknown tool

- **WHEN** the client calls `tools/call` with an unregistered `name`
- **THEN** the server responds with error `-32602` naming the unknown tool
