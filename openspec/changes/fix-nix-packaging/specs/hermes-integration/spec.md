# hermes-integration Specification (delta)

## MODIFIED Requirements

### Requirement: Launch contract

Hermes SHALL launch `pm-mcp` as `${config.services.pm.package}/bin/pm-mcp --dir ${config.services.pm.containerDataDir}` with no hardcoded paths in `hermes.nix`. `containerDataDir` (default `/data/pm`) is the bind-mount target of host `dataDir` inside the container; `PM_DIR` env remains a dev-only override. The flake MUST build from a clean checkout (`vendorHash` real, no placeholders).

#### Scenario: Server starts with data present

- **WHEN** Hermes spawns the `pm` MCP server
- **THEN** the server initializes against the configured container data dir and answers `tools/list`

#### Scenario: Clean build

- **WHEN** running `nix build .#pm` on a fresh clone
- **THEN** the build succeeds and both binaries respond to `--version` with the ldflags-baked version

### Requirement: Binary observability

Both `pm` and `pm-mcp` SHALL report their version (`--version`, baked via `-ldflags -X main.Version`) so the deployed revision is identifiable from the host.

#### Scenario: Version identifies the deploy

- **WHEN** the operator runs `pm --version` on the host
- **THEN** the output names the exact deployed revision
