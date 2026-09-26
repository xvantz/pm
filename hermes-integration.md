# Интеграция PM с Hermes Agent

## Подключение флейка

В корневом `flake.nix` (dotfiles):

```nix
{
  inputs = {
    # ... остальные inputs ...

    pm.url = "git+https://git.827482.xyz/xvantz/pm.git";
  };

  outputs = { nixpkgs, pm, ... } @ inputs: {
    # ...
        inputs.pm.nixosModules.default
        {
          services.pm = {
            enable = true;
            dataDir = "/home/xvantz/Documents/pm";  # хост-путь, по умолчанию
            # containerDataDir = "/data/pm";         # путь ВНУТРИ контейнера Hermes, по умолчанию
          };
        }
  };
}
```

## Настройка MCP сервера в Hermes

В `modules/system/hermes/hermes.nix`:

```nix
{ config, ... }: {
  services.hermes-agent.mcpServers.pm = {
    enabled = true;
    command = "${pkgs.writeShellScriptBin "pm-diag" ''
      echo "starting pm-mcp" >&2
      exec ${config.services.pm.package}/bin/pm-mcp --dir ${config.services.pm.containerDataDir}
    ''}/bin/pm-diag";
  };
}
```

Правила:

- `dataDir` - путь на хосте (bind-mount источник), `containerDataDir` - путь внутри контейнера (mount target). Не путать.
- Wrapper пишет в stderr максимум одну строку на старт. Полные дампы окружения (`env`, `PATH`) запрещены: раздувают `mcp-stderr.log` и светят токены.
- stdout строго для JSON-RPC фреймов.

После `nixos-rebuild switch`:

```bash
sudo systemctl restart hermes-agent
```

Проверка:

```bash
pm --version        # версия пакета с хоста
pm-mcp --version    # то же для MCP-сервера
```

В Hermes появятся инструменты с префиксом `mcp_pm_` (14 шт.):
- `list_projects`, `get_project`, `add_project`
- `add_step`, `start_step`, `review_step`, `done_step`
- `add_blocker`, `resolve_blocker`
- `add_decision`, `get_briefing`
- `list_steps`, `list_blockers`, `list_decisions`
