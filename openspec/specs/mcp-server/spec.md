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

The server SHALL expose the tool set registered in `RegisterPMTools` (`internal/mcp/tools.go`): the live `tools/list` response IS the catalogue, and no tool count is hardcoded in this spec. Every entry SHALL carry `name`, `description` and `inputSchema`. Integrity invariants (non-empty set, unique names, every `inputSchema.type == "object"`) are asserted by `TestToolsList_AllSchemasAreObjects`, which runs against all registered tools: newly registered tools are covered without spec edits.

#### Scenario: List tools

- **WHEN** an initialized client sends `tools/list`
- **THEN** the server returns all registered tools with `name`, `description` and `inputSchema`

#### Scenario: New tool needs no catalogue edit

- **WHEN** a tool is added to `RegisterPMTools` with a valid schema and unique name
- **THEN** `tools/list` includes it and all spec validations still pass unchanged

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

### Requirement: Bulk close tool

The `close_project` tool SHALL bulk-close a finished project in one call: force-complete every non-done step, set status `completed` with `CompletedAt`, and record a `Closed: <reason>` decision. Input schema is `{"type": "object"}` with required `project_id` and optional `reason`. For finished work it replaces N lifecycle calls; for active work the strict lifecycle stays.

#### Scenario: Close with reason

- **WHEN** the client calls `close_project` with a valid `project_id` and `reason`
- **THEN** all open steps become `done`, status is `completed`, and a `Closed:` decision carries the reason

#### Scenario: Close already completed

- **WHEN** the client closes a project whose status is already `completed`
- **THEN** the call fails with an `already completed` error

### Requirement: Delete tool

The `delete_project` tool SHALL expose the existing trash move over MCP: the project leaves the list and lands in `.trash`, recoverable via `trash restore`. For finished work `close_project` is preferred over delete.

#### Scenario: Delete moves to trash

- **WHEN** the client calls `delete_project` with a valid `project_id`
- **THEN** the project disappears from the list and appears in trash
