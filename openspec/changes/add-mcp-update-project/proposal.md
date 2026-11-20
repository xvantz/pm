# Proposal: add-mcp-update-project

## Why

2026-11-20, живой случай: агент не смог перевести проект #17 из статуса
`idea` в `active` через MCP — инструмента для этого нет вообще. Пришлось
обойти MCP и стучать в `PATCH /api/projects/{ref}` напрямую, то есть мимо
санкционированного интерфейса и без согласования.

Вторая дырка того же класса вскрылась там же: в проекте и в шаге остался
старый вендор в заголовке (`Купить и настроить GL.iNet роутер`), а
переименовать нечем. Пришлось пересоздавать проект (#4 → #17): сменились
номер, идентификатор и пропала история шагов. Причина одна — MCP не
покрывает update-операции, хотя store их умеет (`SaveProject`), а API уже
принимает `PATCH` с `goal/status/tags`. Не хватает только `title`, и его
отсутствие делает remote-режим опасным: заголовок меняется в памяти и
молча не сохраняется.

## What Changes

- Новый MCP-инструмент `update_project`: принимает `project_id` и хотя бы
  одно из `title`, `goal`, `tags`, `status`; возвращает обновлённый проект
  и подсказку следующего шага.
- `PATCH /api/projects/{ref}` принимает `title`. Пустая строка — `400`, по
  тому же правилу, что `ErrEmptyTitle` при создании.
- `apistore.SaveProject` передаёт `title` в `PATCH`. Без этого поле
  меняется в памяти клиента и теряется на диске — текущее поведение.

## Non-goals

- Не чиним здесь красный `validate --all --strict` на baseline: проверено на
  `origin/main` (2026-11-20) версиями CLI 1.15.0, 2.0.0, 2.1.0 и 2.1.1 — 5
  спек падают на предупреждении «Requirement text is very long (>500
  characters)» (`cli-lifecycle`, `project-store`, `serve-api`,
  `store-git-backup`, `mcp-server`). CI зелёный только потому, что его
  последний прогон старше текущего `@latest`: версия CLI в `ci.yml` не
  запинена. Лечится отдельным изменением (разбить длинные требования +
  запинить версию CLI), иначе этот change утонет в чужом долге.
- `update_step` (переименование шага) — отдельное изменение: нужен новый
  роут `PATCH /api/projects/{ref}/steps/{step}` и метод стора, сейчас есть
  только действия `start/review/done`.
- Отдельный инструмент `set_status`: статус — обычное поле в
  `update_project`, дублировать его не надо.
- Не трогаем доменные переходы шагов, `close_project` и правила
  консента при закрытии.

## Impact

- Affected specs: `mcp-server` (ADDED: инструмент обновления проекта),
  `serve-api` (MODIFIED: `Endpoints mirror the store` — в `PATCH` добавлен
  `title`; остальные строки требования без изменений).
- Affected code: `internal/mcp/tools.go`, `internal/mcp/update_project_test.go`
  (новый), `internal/api/server.go`, `internal/api/patch_title_test.go`
  (новый), `internal/apistore/apistore.go`.
