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
- [x] P9 MCP: нет пагинации
  **Done:** исходный ченж feat-mcp-paging-briefing закрыт как superseded и разложен.
  Пагинация оказалась неверной рамкой: замер показал дамп 2 195 B, листать нечего.
  Вместо нее feat-mcp-bounded-reads, PR #27: три размера чтения (сводка 429 B,
  краткий список, get_step), detail:true как escape hatch. String ID вынесен
  отдельно: коммит c8c938f, ID стал json.RawMessage с эхом байт-в-байт.
- [ ] P10 Doctor локальный
  **Moved:** ченж fix-env-doctor PM_DIR vs путь демона (расщепление не покрыто)
- [x] P11 Briefing вообще без спеки
  **Done:** ченж chore-briefing-spec, PR #26. Capability briefing: 8 требований,
  14 сценариев, все доказаны тестами. По пути найден баг: нечитаемая date
  откатывала вычисление, но оставляла мусор в ответе.
