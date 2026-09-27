# hermes-integration Specification (delta)

## ADDED Requirements

### Requirement: Wrapper stderr budget

The `pm-diag` wrapper SHALL write at most one diagnostic line to stderr per startup (server name and version). Full environment dumps (`env`, `PATH`) are forbidden: they bloat `mcp-stderr.log` (observed 147MB) and risk leaking secrets into plaintext logs.

#### Scenario: Restart storm stays quiet

- **WHEN** Hermes restarts the `pm` MCP server 10 times in a row
- **THEN** `mcp-stderr.log` grows by at most ~10 lines and contains no `*_TOKEN` values
