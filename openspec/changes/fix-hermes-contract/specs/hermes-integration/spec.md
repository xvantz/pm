# hermes-integration Specification (delta)

## MODIFIED Requirements

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
