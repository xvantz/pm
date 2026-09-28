# Hermes Integration Specification

## Purpose

Contract between the `pm` Nix package (`flake.nix`, `nixosModules.default` with `services.pm`: `enable`, `package`, `dataDir`) and Hermes Agent (`mcpServers.pm` in `hermes.nix`): how `pm-mcp` is launched, where data lives, and stderr discipline. Reference doc: `hermes-integration.md`.

## Requirements

### Requirement: Launch contract

Hermes SHALL launch `pm-mcp` as `${config.services.pm.package}/bin/pm-mcp` with no hardcoded paths and no data volume mounts in `hermes.nix`. The container needs only address (`env.PM_API`, default serve convention `http://127.0.0.1:8472`) + token (`env.PM_TOKEN`, from the same sops `pm_env` secret as the daemon). `dataDir`/`PM_DIR` is the host path and only the daemon reads it; `--dir` exists solely on `pm serve`. The flake MUST build from a clean checkout (`vendorHash` real, no placeholders).

#### Scenario: Server starts with data present

- **WHEN** Hermes spawns the `pm` MCP server
- **THEN** it answers `tools/list` against daemon-held data, with no volume mounts and no `--dir` in its launch command, authenticated by `PM_TOKEN`

#### Scenario: Clean build

- **WHEN** running `nix build .#pm` on a fresh clone
- **THEN** the build succeeds and both binaries respond to `--version` with the ldflags-baked version

#### Scenario: No mounts in launch command

- **WHEN** auditing `hermes.nix` for the `pm` server block
- **THEN** no data volume mount and no `--dir` flag is present; only `PM_API` and `PM_TOKEN` env

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

### Requirement: Wrapper stderr budget

The `pm-diag` wrapper SHALL write at most one diagnostic line to stderr per startup (server name and version). Full environment dumps (`env`, `PATH`) are forbidden: they bloat `mcp-stderr.log` (observed 147MB) and risk leaking secrets into plaintext logs.

#### Scenario: Restart storm stays quiet

- **WHEN** Hermes restarts the `pm` MCP server 10 times in a row
- **THEN** `mcp-stderr.log` grows by at most ~10 lines and contains no `*_TOKEN` values
