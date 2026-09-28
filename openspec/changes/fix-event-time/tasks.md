# Tasks: fix-event-time (draft, DoD уточнить при старте)

## Phase 1: Types + store

- [ ] 1.1 `NowISO` → RFC3339, миграция чтения старых `YYYY-MM-DD`
- [ ] 1.2 Тесты сериализации обоих форматов

## Phase 2: Consumers

- [ ] 2.1 Briefing на точных метках (today/week)
- [ ] 2.2 Сортировки по строкам дат → по времени
- [ ] 2.3 `go test ./...`, README про формат
