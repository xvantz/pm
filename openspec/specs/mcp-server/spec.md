# MCP Server Specification

## Purpose

`pm-mcp` exposes Project Memory operations as MCP tools over stdio transport so LLM agents can read, create and update project data. Source: `cmd/pm-mcp/main.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`.

## Requirements

### Requirement: Stdio JSON-RPC transport

The server SHALL communicate over stdio using JSON-RPC 2.0 messages with Content-Length framing on both read (`messageReader.readMessage`) and write (`writeMessage`).

#### Scenario: Initialize handshake

- **WHEN** the client sends `initialize` with any params
- **THEN** the server responds with `protocolVersion: "2024-11-05"`, empty `capabilities.tools` and `serverInfo` (name, version), without requiring prior state

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

Every tool inputSchema SHALL be a JSON Schema object. `list_projects` uses `{}` (empty schema, no `type` field); all other tools declare `"type": "object"` with `properties` and `required`.

#### Scenario: Call list_projects

- **WHEN** the client calls `list_projects` with `{}` arguments
- **THEN** the handler returns the project list as text content

### Requirement: Tool call envelope

`tools/call` SHALL dispatch by `name`, pass `arguments` to the handler, and wrap the handler's string result as `content: [{type: "text", text}]`. Unknown tool names SHALL return `-32602`, handler failures SHALL return `-32603`.

#### Scenario: Unknown tool

- **WHEN** the client calls `tools/call` with an unregistered `name`
- **THEN** the server responds with error `-32602` naming the unknown tool
