# Интеграция PM с Hermes Agent

## 1. MCP сервер

Добавить в `/data/.hermes/config.yaml` (в секцию `mcp_servers`):

```yaml
  pm:
    command: /projects/pm/pm-mcp
    args: []
    env:
      PM_DIR: /projects/pm/pm/projects
    enabled: true
```

После чего перезапустить Hermes:

```bash
sudo systemctl restart hermes-agent
```

После перезапуска в Hermes появятся инструменты с префиксом `mcp_pm_`:
- `mcp_pm_list_projects`
- `mcp_pm_get_project`
- `mcp_pm_add_project`
- `mcp_pm_add_step`
- `mcp_pm_start_step`
- `mcp_pm_review_step`
- `mcp_pm_done_step`
- `mcp_pm_add_blocker`
- `mcp_pm_resolve_blocker`
- `mcp_pm_add_decision`
- `mcp_pm_get_briefing`
- `mcp_pm_list_steps`
- `mcp_pm_list_blockers`
- `mcp_pm_list_decisions`

## 2. NixOS модуль

Если Hermes управляется через NixOS, добавить в `services.hermes-agent.mcpServers`:

```nix
services.hermes-agent = {
  mcpServers.pm = {
    command = "${pkgs.pm}/bin/pm-mcp";
    env.PM_DIR = "/home/xvantz/pm/projects";
  };
};
```

Где `pkgs.pm` — пакет из flake.nix этого проекта.
