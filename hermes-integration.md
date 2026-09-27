# PM + Hermes Agent integration

## Flake input

In the root `flake.nix` (dotfiles):

```nix
{
  inputs = {
    # ... other inputs ...

    pm.url = "git+https://git.827482.xyz/xvantz/pm.git";
  };

  outputs = { nixpkgs, pm, ... } @ inputs: {
    # ...
        inputs.pm.nixosModules.default
        {
          services.pm = {
            enable = true;
            dataDir = "/home/xvantz/Documents/pm";  # host path, the default
            environmentFile = config.sops.secrets.pm_env.path;
            # listenAddr = "127.0.0.1:8472";         # the default
          };
        }
  };
}
```

Declare the secret next to the service (same pattern as `hermes_env`):

```nix
sops.secrets.pm_env = {
  owner = "xvantz";
  restartUnits = [ "pm-serve.service" ];
};
```

And add `PM_TOKEN=<token>` to `secrets.yaml` via sops on the host
(the age key lives outside containers, so this step is host-only).

This creates the `pm-serve` systemd service (`restart=always`): the single
writer owning the YAML. Token reaches the daemon via `EnvironmentFile`
(sops-managed env file, same convention as `hermes_env`) —
never through the nix store or unit text. Enabling the service without
`environmentFile` fails the build with an assertion telling exactly
what to add.

## MCP server in Hermes

In `modules/system/hermes/hermes.nix`:

```nix
{ config, ... }: {
  services.hermes-agent.mcpServers.pm = {
    enabled = true;
    command = "${config.services.pm.package}/bin/pm-mcp";
    env.PM_API = "http://127.0.0.1:8472";
    env.PM_TOKEN = "\${PM_TOKEN}";
  };
}
```

No data volume mounts. No `--dir`. The container needs only address + token.

Rules:

- `dataDir` is the host path and only the daemon reads it. Nothing else mounts it.
- `pm-diag` wrapper (if used) prints at most one stderr line on start. Full
  environment dumps (`env`, `PATH`) are forbidden: they bloat `mcp-stderr.log`
  and leak tokens.
- stdout is strictly for JSON-RPC frames.

After `nixos-rebuild switch`:

```bash
sudo systemctl restart hermes-agent
```

Checks:

```bash
pm --version        # package version from the host
pm-mcp --version    # same for the MCP server
curl localhost:8472/healthz
curl -H "Authorization: Bearer $PM_TOKEN" localhost:8472/api/projects
```

Hermes gains the tools with the `mcp_pm_` prefix (13 total):
- `list_projects`, `get_project`, `add_project`
- `add_step`, `start_step`, `review_step`, `done_step`
- `add_blocker`, `resolve_blocker`
- `add_decision`, `get_briefing`
- `list_steps`, `list_blockers`, `list_decisions`
