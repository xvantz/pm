# Transport spike: decision record

## Verified 2026-09-15 (read of `internal/mcp/server.go` on main + live `tools/list` bytes)

- `writeMessage` (server.go:198) emits NDJSON: one JSON object per line, explicitly NO `Content-Length` headers ("the MCP Python SDK reads stdout").
- `messageReader.readMessage` (server.go:233) is dual-mode: plain NDJSON lines plus `Content-Length:` framed input (used by some clients and tests).
- Live `tools/list` output from the fixed code is a single JSON line; all 14 `inputSchema` carry `"type": "object"`.

## Decision: variant A already holds, no rewrite

Neither rollback nor SDK migration is needed. The code on main already matches the Hermes Python MCP client. What lagged was the test helper (`readMCPResponse` expected Content-Length) plus stale `handleMessage` call sites - both fixed on branch `fix/mcp-schema-transport`.

## Remaining risk (not this change)

Hand-rolled `ID *int` still breaks string-ID clients; `ping` / `notifications/cancelled` unhandled. Accept for now; revisit only if Hermes updates break the handshake (tracked, not scheduled).
