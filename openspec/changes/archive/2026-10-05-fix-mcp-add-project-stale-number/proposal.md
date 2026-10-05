# Proposal: fix-mcp-add-project-stale-number

## Why

Прод-баг с живым репродом (05.10.2026, remote mode `PM_API=http://127.0.0.1:8472`):
`add_project` через MCP ответил `Project #10`, а прямой `GET /api/projects/<uuid>`
показал `"number": 14`. Следующий `add_step {project_id: "10"}` дал
`404 project "10" not found`. Три подряд 404 уронили gateway circuit breaker
(парковка MCP сервера pm на ~60 сек), пострадали все последующие PM вызовы сессии.
Trash в момент репрода: #10-#13 (удаленные тестовые проекты 27-28.09.2026),
живые номера 1-9, 14.

Цепочка расхождения:

1. `handleAddProject` (`internal/mcp/tools.go:604-654`) берет `st.NextNumber()`,
   кладет в `p.Number`, печатает `p.Number` в confirmation.
2. В remote mode `st` это `apistore.Store`: `NextNumber()` advisory, max+1 по
   живому `ListProjects` (trash исключен). При живых 1-9 дает 10.
3. `apistore.SaveProject()` (`apistore.go:80-89`) шлет серверу только
   title/goal/tags/id, номер не передает; ответ `CreateProject`
   (`client.go:124`, возвращает `*types.Project` с реальным номером) отбрасывается.
4. Сервер (`server.go:174-195`) назначает номер сам из файлового счетчика
   (`filestore.go:204-280`), который считает все advances включая удаленные
   в trash. Реальный номер 14.
5. Итог: confirmation врет после ЛЮБОГО удаления, не только при race.
   Комментарий в `apistore.go:8-10` признавал только concurrent-случай,
   но trash гарантирует расхождение навсегда.

CLI (`cmdProjectCreate`, `internal/cli/project.go:82-88`) уже перечитывает
проект после save и печатает серверный номер (спека
`cli-lifecycle / Creation echoes the server-assigned number`). MCP хендлер
этот паттерн не повторял - спеки `mcp-server` такого требования не содержат.
Асимметрия CLI/MCP и есть дефект.

## What Changes

- **MCP хендлер перечитывает после save**: `handleAddProject` после
  `SaveProject` читает проект по стабильному UUID (`GetProject`, fallback
  `ResolveProject`) и печатает вычитанный `Number`. При ошибке чтения -
  громкая ошибка `resolve created project`, а не вранье с advisory номером.
  Без изменения интерфейса `Store`, один лишний read, работает на всех backend.
- **Строка Next несет UUID, а не номер**: `Next: add_step {project_id: "<uuid>"}`.
  UUID назначается client-side до save и никогда не stale; номер в заголовке
  `Project #N` при этом тоже корректен (вычитанный).
- **Комментарий apistore переписан**: было "concurrent creates may print
  a stale number", стало - caller обязан перечитывать; оба caller
  (MCP + CLI) это делают.
- **Регресс-тест `internal/mcp/add_project_stale_test.go`**: `trashGapStore`
  поверх `MockStore` (живой max 6, серверный счетчик 14) - падает на старом
  коде (печатает #7), зелен после фикса (#14 + `add_step` по напечатанному
  UUID с первого раза + второй проект #15).

## Non-goals

- Не перенумеровываем существующие проекты.
- Не переиспользуем номера из trash (по дизайну не занимают повторно).
- Не меняем поведение FileStore напрямую (local mode не затронут - там один
  счетчик на read и save).
- Не трогаем gateway circuit breaker (сработал штатно).

## Impact

- Affected specs: `mcp-server` (ADDED: confirmation несет серверный номер,
  Next - UUID).
- Affected code: `internal/mcp/tools.go` (re-read + UUID в Next),
  `internal/apistore/apistore.go` (комментарий),
  `internal/mcp/add_project_stale_test.go` (новый регресс-тест).
- Соседи проверены grep `number` по `openspec/specs/`: `project-store /
  Monotonic project numbers` (монотонность, не затронута - дырки от trash
  были и будут), `cli-lifecycle / Creation echoes the server-assigned number`
  (MCP теперь повторяет тот же паттерн), `serve-api` create (числа остаются
  server-assigned, без изменений). Правок в них не требуется.
