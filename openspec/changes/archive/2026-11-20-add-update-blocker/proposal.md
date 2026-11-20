# Proposal: add-update-blocker

## Why

Тот же класс, что закрытый `add-mcp-update-project`: текст блокера нельзя
поменять ни через MCP, ни через API. Если причина устарела («жду посылку»,
а посылка приехала, но железо ещё не настроено), остаётся только удалить
блокер и создать заново — с новым id, и все ссылки на старый id
превращаются в мусор. Работа с доской превращается в пересоздание.

Хуже: `apistore.SaveBlocker` в remote-режиме умел только два действия —
создать и «resolve». Правка title/reason не доезжала до демона вообще, то
есть локальный `SaveBlocker` с новым текстом молча ничего не менял.

## What Changes

- `PATCH /api/projects/{ref}/steps/{step}/blockers/{blk}` `{title?, reason?}`:
  пустой `title` — `400`, slug-коллизия внутри шага — `409`, неизвестный
  блокер — `404`, вызов без полей — `400`. Id блокера не меняется.
- `client.PatchBlocker` и `apistore.SaveBlocker`: правка текста уходит в
  демон; resolve-переход остаётся отдельной веткой, как и был.
- MCP-инструмент `update_blocker` (`project_id`, `step_id`, `blocker_id`,
  и хотя бы одно из `title`/`reason`).

## Non-goals

- Решения (`decisions`) остаются иммутабельными: это запись «почему так
  решили», её правка превращает журнал в черновик. Меняется только текст
  блокеров и шагов.
- Статус блокера не двигаем через patch: для него есть отдельный роут
  `.../resolve`, у которого своя семантика (снятие блокера не должно
  теряться в общем апдейте).

## Impact

- Affected specs: `mcp-server` (ADDED: инструмент правки блокера),
  `serve-api` (MODIFIED: `Endpoints mirror the store` — добавлен PATCH
  блокера).
- Affected code: `internal/api/server.go`, `internal/api/patch_blocker_test.go`
  (новый), `internal/client/client.go`, `internal/apistore/apistore.go`,
  `internal/apistore/blocker_edit_test.go` (новый),
  `internal/mcp/tools.go`, `internal/mcp/update_blocker_test.go` (новый).
