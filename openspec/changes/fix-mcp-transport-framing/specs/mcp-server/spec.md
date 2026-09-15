# mcp-server Specification (delta)

## MODIFIED Requirements

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
