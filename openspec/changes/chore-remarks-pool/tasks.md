# Tasks: chore-remarks-pool (triage pool, DoD ставится при разборе пункта)

## Confirmed drift (спека врет, чинить ченжем)

- [x] P1 mcp-server catalogue: "exactly 14 tools" при фактических 16 (close_project, delete_project)
  **Done:** PR #19, каталог переписан как derived, тест уникальности имен
- [x] P2 hermes-integration Launch contract: описывает containerDataDir/bind-mount мир, код уже remote-only
  **Done:** PR #20, MODIFIED под remote-only, док 13 → 16
- [ ] P3 cli-lifecycle Trash: "CLI-only, no MCP tool" при живом delete_project в MCP
  **Deferred:** соседом в feat-mcp-trash-restore (MODIFIED Trash в матрицу при его реализации), закрывается его архивом

## Design critique (нужно решение keep/change)

- [x] P4 CloseProject doneит шаги с висящими блокерами (спека фиксирует bypass как intended; вопрос: резолвить блокеры при close?)
  **Done:** вердикт keep. Инвариант проверен в коде: briefing собирает блокеры только под StatusActive (briefing.go switch), completed уходят в completedSec без подсчетов; list_blockers только per-project. Закрытие = выполнено, блокеры история. Guard: правило соседей в скилле (правка briefing обязана грепнуть инвариант)
- [ ] P5 Create печатает advisory номер вместо серверного (врет при гонке)

## Coverage gaps (спеки молчат)

- [ ] P6 Гранулярность времени: NowISO только дата, порядок внутри дня теряется
- [ ] P7 Slug правила не специфицированы (набор символов, длина, скрытые файлы)
- [ ] P8 Модуль не раздает PM_API (смена listenAddr роняет клиентов)
- [ ] P9 MCP: нет пагинации get_project, string ID не поддерживаются
- [ ] P10 Doctor локальный PM_DIR vs путь демона (расщепление не покрыто)
- [ ] P11 Briefing вообще без спеки (capability файл отсутствует)
