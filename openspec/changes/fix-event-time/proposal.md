# Proposal: fix-event-time

## Why

`NowISO` пишет только дату (P6 пула). Все UpdatedAt внутри дня одинаковые: порядок событий теряется, брифинг считает любой touch шагом "сделанным сегодня", `blockerDaysAlive` грубый до дней. Для памяти проектов это слепота к intraday истории.

## What Changes

- Время в RFC3339 для CreatedAt/UpdatedAt/Date во всех сущностях; дата отдельно для UI где нужна.
- Брифинг считает по точным меткам (today/week корректно при нескольких событиях в день).
- Миграция существующих YAML: дата без времени трактуется как начало дня UTC.

## Impact

- Affected specs: `project-store` (MODIFIED: формат меток), новая capability под брифинг? Брифинг спеки нет (P11 едет отдельно).
- Affected code: `internal/types`, `internal/store`, `internal/briefing`, `internal/api`.
- Соседи грепнуть: все сравнения `UpdatedAt == date`, сортировки по строкам дат.
