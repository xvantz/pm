# Hermes Integration Specification

## Purpose

Contract between the `pm` Nix package (`flake.nix`, `nixosModules.default` with `services.pm`: `enable`, `package`, `dataDir`) and Hermes Agent (`mcpServers.pm` in `hermes.nix`): how `pm-mcp` is launched, where data lives, and stderr discipline. Reference doc: `hermes-integration.md`.

## Requirements

### Requirement: Launch contract

Hermes SHALL launch `pm-mcp` as `${config.services.pm.package}/bin/pm-mcp --dir /data/pm` via the `pm-diag` wrapper script. The data directory inside the Hermes container is `/data/pm`, bind-mounted from the host data dir.

#### Scenario: Server starts with data present

- **WHEN** Hermes spawns the `pm` MCP server
- **THEN** the server initializes against `/data/pm` and answers `tools/list`

### Requirement: Stderr discipline

The wrapper and the server SHALL log only single-line startup/shutdown notices to stderr. stdout is reserved for JSON-RPC frames; anything else on stdout breaks the session.

#### Scenario: Clean startup lines

- **WHEN** Hermes starts the `pm` MCP server
- **THEN** stderr gains at most one startup line and stdout carries only JSON-RPC frames

### Requirement: Binary observability

Both `pm` and `pm-mcp` SHALL report their version (`--version`, baked via `-ldflags -X main.Version`) so the deployed revision is identifiable from the host.

#### Scenario: Version identifies the deploy

- **WHEN** the operator runs `pm --version` on the host
- **THEN** the output names the exact deployed revision
