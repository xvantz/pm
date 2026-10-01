# Tasks: chore-remarks-pool (triage pool, DoD ставится при разборе пункта)

## Confirmed drift (спека врет, чинить ченжем)

- [x] P1 mcp-server catalogue: "exactly 14 tools" при фактических 16 (close_project, delete_project)
  **Done:** PR #19, каталог переписан как derived, тест уникальности имен
- [x] P2 hermes-integration Launch contract: описывает containerDataDir/bind-mount мир, код уже remote-only
  **Done:** PR #20, MODIFIED под remote-only, док 13 → 16
- [x] P3 cli-lifecycle Trash: "CLI-only, no MCP tool" при живом delete_project в MCP
  **Done:** ченж feat-mcp-trash-restore, PR #29. Trash открыт в MCP (list/restore),
  wipe остался CLI-only; плюс бэкапы на все удаления. Baseline вычищен.

## Design critique (нужно решение keep/change)

- [x] P4 CloseProject doneит шаги с висящими блокерами (спека фиксирует bypass как intended; вопрос: резолвить блокеры при close?)
  **Done:** вердикт keep. Инвариант проверен в коде: briefing собирает блокеры только под StatusActive (briefing.go switch), completed уходят в completedSec без подсчетов; list_blockers только per-project. Закрытие = выполнено, блокеры история. Guard: правило соседей в скилле (правка briefing обязана грепнуть инвариант)
- [x] P5 Create печатает advisory номер вместо серверного (врет при гонке)
  **Done:** PR #21, re-read по ID после save, живой рейс 3 параллельных create (11,12,13 резолвятся)

## Coverage gaps (спеки молчат)

- [x] P6 Гранулярность времени
  **Done:** ченж fix-event-time. Тип `types.Timestamp` (RFC3339 UTC), legacy-читается
  как полночь UTC, три тихие поломки в брифинге закрыты, зона дня = зона демона,
  doctor считает legacy/битые метки, миграция автоматическая при следующей записи
- [x] P7 Slug правила
  **Done:** ченж fix-slug-rules, PR #30. Allowlist на рунах + кап 200 байт.
  Claim про скрытые файлы опровергнут: точка не может появиться в выводе.
- [ ] P8 Модуль не раздает PM_API
  **Moved:** ченж fix-env-doctor (смена listenAddr роняет клиентов)
- [ ] P9 MCP: нет пагинации
  **Moved:** ченж feat-mcp-paging-briefing get_project, string ID не поддерживаются
- [ ] P10 Doctor локальный
  **Moved:** ченж fix-env-doctor PM_DIR vs путь демона (расщепление не покрыто)
- [ ] P11 Briefing вообще без спеки
  **Moved:** ченж feat-mcp-paging-briefing (capability файл отсутствует)
