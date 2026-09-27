# Why

Три скрытых дефекта стора: (1) `ResolveProject` сканирует все проекты с диска дважды (числом и префиксом - `filestore.go:70-107`), каждый `tools/call` с `project_id` - это 2 полных обхода; (2) `ListProjects` молча скипает битые проекты через `slog.Warn` - потеря данных видна только в логах, `doctor` о них не знает; (3) `next_number` без атомарности на инкремент: два параллельных `add_project` могут выдать один номер, flock покрывает только запись файлов. Плюс `pm doctor` в контейнере всегда красный - ищет сторедж в дефолтном cwd вместо реального `--dir`.

# What Changes

- `ResolveProject`: один проход с построением индекса (number + id + prefix), один скан на вызов.
- Битые проекты: `ListProjects` возвращает их как `unreadable` записи (dir + error), `doctor` их показывает и предлагает `trash`/восстановление из бэкапа (см. feat-mcp-trash-restore). Молчаливый skip уходит.
- `next_number`: инкремент под тем же flock, что и создание проекта (атомарно).
- `doctor`: уважает `--dir`/`PM_DIR`, проверяет counter, бэкапы и битые проекты; exit code nonzero при проблемах (для CI).

# Impact

- Affected specs: `project-store` (MODIFIED: резолв, битые проекты, счетчик), `cli-lifecycle` (MODIFIED: doctor).
- Affected code: `internal/store/filestore.go`, `internal/cli/doctor.go`, тесты стора.
- Поведенческое изменение: `ListProjects` больше не молчит о битых данных - клиенты увидят `unreadable` записи. Это intended (намеренно).
