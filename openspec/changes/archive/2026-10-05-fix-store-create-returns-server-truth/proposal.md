# Proposal: fix-store-create-returns-server-truth

## Why

`POST /api/projects` уже возвращает `201` с полным проектом (верный номер
внутри), `client.CreateProject` его парсит - а `apistore.SaveProject` ответ
выкидывает (`_, cerr := s.c.CreateProject(...)`). Причина не в демоне, а в
интерфейсе: `Store.SaveProject(p types.Project) error` возвращает только
ошибку, ему некуда положить серверную правду. Интерфейс проектировали под
прямую запись в файлы, демона прикрутили позже адаптером.

Текущий фикс (re-read по UUID в `handleAddProject` и `cmdProjectCreate`)
корректен, но это костыль на каждый create: два round-trip вместо одного,
и каждый новый caller creation-пути обязан помнить про перечитку. Забывший
caller снова напечатает advisory номер. Правильно - дать созданию голос:
метод, возвращающий созданное.

## What Changes

- **Новый метод в интерфейсе `Store`: `CreateProject(title, goal, tags, id)
  (types.Project, error)`**. Семантика: создает проект со статусом idea,
  номер назначает хранилка (файловый счетчик / серверный mutex / max+1 в моке),
  пустой id значит "сгенерируй", непустой UUIDv7 - закрепи (409 на коллизии,
  400 на мусоре, как сегодня у демона).
- **Три реализации**: `FileStore` (счетчик под существующим локом - та же
  логика, что сегодня в `server.go:handleProjectCreate`, спускается внутрь),
  `MockStore` (max+1), `apistore` (делегирует `client.CreateProject` и
  ВОЗВРАЩАЕТ ответ вместо `_`).
- **Миграция двух caller**: `cmdProjectCreate` и `handleAddProject` зовут
  `CreateProject` и печатают вернувшееся - re-read уходит, поведение
  (заголовок `Project #N`, UUID в `Next:`) не меняется, trash-gap тест
  остается зеленым без перечитки.
- **`NextNumber`/`AdvanceNextNumber`**: из creation-пути уходят; на интерфейсе
  помечаются deprecated до миграции всех caller, затем удаляются (или остаются
  только у `FileStore` как деталь реализации - решает исполнитель, граница
  записана в задачах).

## Non-goals

- Не перенумеровываем существующие проекты, номера из trash не переиспользуем.
- Не меняем wire-контракт `POST /api/projects` (он уже правильный).
- Не трогаем update-путь `SaveProject` (тач timestamp, статусы, голы) и его
  сигнатуру - все 15+ вызовов остаются как есть.
- Не трогаем gateway circuit breaker.
- Поведение `ServeAPI` снаружи не меняется: тот же `201`, те же коды ошибок.

## Impact

- Affected specs: `project-store` (ADDED: создание возвращает серверную правду).
- Affected code: `internal/store/store.go` (интерфейс + deprecated-метки),
  `internal/store/filestore.go`, `internal/store/mock.go`,
  `internal/apistore/apistore.go`, `internal/cli/project.go`,
  `internal/mcp/tools.go`, `internal/mcp/add_project_stale_test.go`
  (тот же тест, без re-read в коде), плюс все тесты, строящие кастомные
  реализации `Store` (компилятор перечислит).
- `cli-lifecycle / Creation echoes the server-assigned number` и новый
  `mcp-server / Project creation echoes the server-assigned number` остаются
  верны дословно - меняется механизм, не поведение. Правок в них не требуется.
- Риск: смена интерфейса ломает внешние реализации `Store` вне репо (если
  есть). Митигация: deprecated-период `NextNumber` и релиз-ноут в PR.
