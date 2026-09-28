# serve-api Specification (delta)

## MODIFIED Requirements

### Requirement: Nix options

The flake module SHALL expose `services.pm.listenAddr`
(default `127.0.0.1:8472`) and `services.pm.environmentFile`
(sops env file with `PM_TOKEN`, same convention as `hermes_env`).
An enabled service without `environmentFile` fails the build via
`assertions` with exact fix instructions. `services.pm.user` is REQUIRED
(no default). When set, a
`pm-serve` systemd service is created: `restart=always`,
token via `EnvironmentFile` (never in nix store or unit text),
`PM_DIR` from `services.pm.dataDir`.
The module also provisions `PM_TOKEN` into login shells itself
(`programs.zsh/bash.interactiveShellInit` sourcing the same env file
at runtime): no hand edits to shell configs needed.
The module SHALL also provision `PM_API` derived from `listenAddr`,
so changing the address never orphans clients on the default.

#### Scenario: host enables daemon

- **WHEN** `services.pm = { enable = true; environmentFile = <pm_env path>; }`
- **THEN** after `nixos-rebuild`, `curl localhost:8472/healthz` answers
  and authed `/api/projects` round-trips
- **WHEN** `services.pm = { enable = true; }` without `environmentFile`
- **THEN** the build fails with an assertion naming the missing option

#### Scenario: custom address reaches clients

- **WHEN** `listenAddr` is changed from the default
- **THEN** shells and MCP pick the new address from provisioned `PM_API` with no hand edits
