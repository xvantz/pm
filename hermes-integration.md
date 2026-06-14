# Интеграция PM с Hermes Agent

## Подключение флейка

В корневом `flake.nix` (dotfiles):

```nix
{
  inputs = {
    # ... остальные inputs ...

    pm = {
      url = "github:xvantz/pm";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { nixpkgs, pm, ... } @ inputs: {
    nixosConfigurations.nixos = nixpkgs.lib.nixosSystem {
      modules = [
        # ... остальные модули ...

        pm.nixosModules.default
        {
          services.pm = {
            enable = true;
            dataDir = "/home/xvantz/Documents/pm";  # по умолчанию
          };
        }
      ];
    };
  };
}
```

## Настройка MCP сервера в Hermes

В `hermes.nix` (или `modules/system/hermes.nix`):

```nix
{ config, ... }: {
  services.hermes-agent = {
    # ... остальные настройки ...

    mcpServers.pm = {
      command = "${config.services.pm.package}/bin/pm-mcp";
      env.PM_DIR = config.services.pm.dataDir;
    };
  };
}
```

После `nixos-rebuild switch`:

```bash
sudo systemctl restart hermes-agent
```

В Hermes появятся инструменты с префиксом `mcp_pm_` (14 шт.):
- `list_projects`, `get_project`, `add_project`
- `add_step`, `start_step`, `review_step`, `done_step`
- `add_blocker`, `resolve_blocker`
- `add_decision`, `get_briefing`
- `list_steps`, `list_blockers`, `list_decisions`
