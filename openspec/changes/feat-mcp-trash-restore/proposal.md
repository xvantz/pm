# Why

Корзина (`trash list` / `trash restore`) существует только в CLI. Агент через MCP может удалить (и теряет путь восстановления без захода в терминал) - асимметрия возможностей. Плюс удаление необратимо по сути: бэкапа перед `trash` нет, кривой `restore` по имени молча берет первое совпадение. Для памяти проектов, где удаление - самая опасная операция, это дыра.

# What Changes

- Новые MCP тулы: `trash_list` (что в корзине) и `trash_restore` (по имени/ID, при неоднозначности - ошибка со списком кандидатов, а не молчаливый первый).
- Перед каждым удалением FileStore пишет бэкап YAML в `_meta/backups/<timestamp>/` (ротация: последние 20).
- `trash restore` в CLI и MCP делят один код пути восстановления.

# Impact

- Affected specs: `mcp-server` (ADDED: 2 тула, 14 -> 16), `project-store` (ADDED: бэкапы).
- Affected code: `internal/mcp/tools.go`, `internal/store/filestore.go`, `internal/cli/trash.go`.
- Зависит от `fix-mcp-list-projects-schema` (новые тулы сразу с корректными схемами).
