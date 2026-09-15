# Why

Строгий lifecycle `add -> start -> review -> done` с обязательным `review` - главная фрустрация агентов: `done_step` на `in_progress` падает, и чтобы закрыть степ приходится делать 3 вызова вместо одного. Архивация проекта превращается в `3N` вызовов. Легитимных кейсов масса: микрозадачи без ревью, принятие работы за агента, импорт завершенных спеко�� из OpenSpec. Запрет обхода должен быть дефолтом, а не кандалами.

# What Changes

- `done_step` (MCP) и `pm step done` (CLI) получают флаг `force`: разрешает переход `todo/in_progress -> done` напрямую, записывая авто-decision `force-completed: <reason>` в проект для аудита.
- Без флага поведение прежнее: `done` не из `review` падает.
- `pm doctor` проверяет наличие force-закрытых степов отдельно не надо - они видны как decision.

# Impact

- Affected specs: `cli-lifecycle` (ADDED: force completion), `mcp-server` (затронут `done_step`, без новой спеки - поведение описано здесь).
- Affected code: `internal/mcp/tools.go` (схема `done_step` + хендлер), `internal/cli/step.go`, `internal/domain/step.go`.
- Обратная совместимость полная: без флага все как было.
