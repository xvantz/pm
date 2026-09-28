# Tasks: chore-remarks-pool (triage pool, DoD ставится при разборе пункта)

## Confirmed drift (спека врет, чинить ченжем)

- [ ] P1 mcp-server catalogue: "exactly 14 tools" при фактических 16 (close_project, delete_project)
- [ ] P2 hermes-integration Launch contract: описывает containerDataDir/bind-mount мир, код уже remote-only
- [ ] P3 cli-lifecycle Trash: "CLI-only, no MCP tool" при живом delete_project в MCP

## Design critique (нужно решение keep/change)

- [ ] P4 CloseProject doneит шаги с висящими блокерами (спека фиксирует bypass как intended; вопрос: резолвить блокеры при close?)
- [ ] P5 Create печатает advisory номер вместо серверного (врет при гонке)

## Coverage gaps (спеки молчат)

- [ ] P6 Гранулярность времени: NowISO только дата, порядок внутри дня теряется
- [ ] P7 Slug правила не специфицированы (набор символов, длина, скрытые файлы)
- [ ] P8 Модуль не раздает PM_API (смена listenAddr роняет клиентов)
- [ ] P9 MCP: нет пагинации get_project, string ID не поддерживаются
- [ ] P10 Doctor локальный PM_DIR vs путь демона (расщепление не покрыто)
- [ ] P11 Briefing вообще без спеки (capability файл отсутствует)
