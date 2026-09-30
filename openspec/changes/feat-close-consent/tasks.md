# Tasks: feat-close-consent (переименован из feat-step-lifecycle-force, DoD на каждой)

Правило 2 нарушено всеми 5 открытыми ченджами — здесь DoD есть на каждой задаче.
Чендж переписан по итогам обсуждения: вместо `done_step --force` (решает
несуществующую боль, ослабляет стопор) — `close_project` требует согласия.


## Phase 1: Store (ClosePlan)

- [x] 1.1 `internal/types`: `ClosePlan` — список шагов к закрытию (ID, title,
  статус) + висящие блокеры (ID, title, reason, step_id)
  **DoD:** `CGO_ENABLED=0 go build ./...` green; тип экспортирован из `types`
  **Результат:** `internal/types/closeplan.go` — `ClosePlan`, `ClosePlanStep`, `ClosePlanBlocker`, методы `StepsToClose`/`BlockersToResolve`/`Empty`. build green.
- [x] 1.2 `internal/store/store.go`: `ClosePlan(projectID) (*ClosePlan, error)`
  в интерфейсе
  **DoD:** `go build ./...` green; все реализации `Store` удовлетворяют
  интерфейсу (`FileStore`, `MockStore`)
  **Результат:** `CloseProject(ref, reason string, confirm bool) (*types.ClosePlan, error)` и `ClosePlan(ref)` в интерфейсе. build green, все три реализации удовлетворяют.
  **Важное отклонение от плана:** confirm добавлен В `store.Store`, а не только в MCP-тул. Причина найдена по коду: MCP работает через `store.Store` → `apistore` → client, поэтому гейт только в туле обходился бы адаптером.
- [x] 1.3 `internal/store/filestore.go` + `mock.go`: реализация `ClosePlan` —
  read-only, ничего не мутирует
  **DoD:** тест: `ClosePlan` на проекте с 3 открытыми степ-ами и 1 блокером
  возвращает 3 шага с их статусами и 1 блокер; после вызова `GetProject`
  возвращает идентичные данные (не изменилось ничего)
  **Результат:** `buildClosePlan` — общий билдер для FileStore и MockStore, чтобы предпросмотр не расходился между бэкендами. 5 тестов в `store_test.go`: без confirm ничего не меняется (статус, шаги, decisions), план read-only, блокеры со StepName.
- [x] 1.4 Юнит-тест: `ClosePlan` на `completed` проекте → пустой план, не ошибка
  **DoD:** `CGO_ENABLED=0 go test ./internal/store/ -count=1` green
  **Результат:** `TestClosePlan_CompletedProjectIsEmptyNotError` — пустой план, не ошибка.

## Phase 2: API (consent на эндпоинте)

- [x] 2.1 `internal/api/server.go` `handleProjectClose`: поле `Confirm bool`;
  без `confirm` → `ClosePlan` в ответе, проект не тронут; с `confirm` и пустым
  reason → 4xx, не тронут
  **DoD:** тест `internal/api/server_test.go`: POST `/close` без `confirm` →
  200 с планом, `GET /api/projects/{ref}` после показывает `active` и
  исходные статусы; POST с `confirm:true` без reason → 422, проект `active`;
  POST с `confirm:true` и reason → 200, `completed`
  **Результат:** `closeProjectReq.Confirm`, ветка plan/apply, `closeProjectResp{Confirmed, Plan, Project}`. Плюс **новый эндпоинт** `GET /api/projects/{ref}/close-plan` — его не было, тест apistore поймал 404. `TestProjectClose` переписан под контракт: план → проект не closed; confirm без reason → 422; confirm с reason → closed; double close → 422.
- [x] 2.2 Клиент `internal/client/client.go`: `ClosePlan` + `Confirm` в `CloseProject`
  **DoD:** `go build ./...` green; сигнатура покрывает оба вызова
  **Результат:** `client.CloseProject(ref, reason, confirm) (*Project, *ClosePlan, error)` + `client.ClosePlan`. Ответ типизирован `closeResponse{Confirmed, Project, Plan}` — не гадание по заполненности.
- [x] 2.3 `internal/apistore/apistore.go`: проброс `ClosePlan`
  **DoD:** `CGO_ENABLED=0 go build ./...` green
  **Результат:** apistore пробросил confirm без изменений. Тесты: `TestRemote_CloseConsentGate`, `TestRemote_ClosePlanDoesNotMutate` — гейт держится по HTTP.

## Phase 3: MCP

- [x] 3.1 `internal/mcp/tools.go` `close_project`: `confirm` в inputSchema;
  без `confirm` — текстовый план (список шагов + блокеры + предупреждение),
  с `confirm` и reason — закрытие
  **DoD:** MCP-тест: вызов без `confirm` возвращает план, `get_project` после
  показывает `active`
  **Результат:** `confirm` в inputSchema, описание тула с пометкой REQUIRES CONSENT, `formatClosePlan` рендерит план для показа человеку.
- [x] 3.2 MCP-тесты: confirm без reason падает; confirm с reason закрывает;
  completed → already completed
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1` green
  **Результат:** 5 тестов в `mcp/close_consent_test.go`: план не закрывает, confirm без reason падает, confirm с reason закрывает, план говорит «does NOT resolve» про блокеры, completed → ошибка.

## Phase 4: CLI

- [x] 4.1 `internal/cli/project.go`: `pm project close <id> <reason>` — reason
  обязателен (пустой → ошибка, дефолт `bulk close` из CLI убран)
  **DoD:** тест: `cmdProjectClose(["1"])` возвращает ошибку; с reason закрывает
  **Результат:** reason обязателен, дефолт `bulk close` убран. Тест поймал **баг**: `cmdProjectClose` разыменовывал `plan`, который apistore возвращает nil на confirm — паника в проде-коде. Исправлено.
- [x] 4.2 Help-текст: одна команда, reason обязателен
  **DoD:** `pm help` содержит обновлённую строку close
  **Результат:** `pm project close <id> <reason>` в help, с пометкой что reason обязателен.

## Phase 5: Проверка

- [x] 5.1 `README.md`: раздел про закрытие — план, согласие, reason; почему
  CLI в одну команду, а агент в два
  **DoD:** раздел есть, пример обоих вызовов
  **Результат:** README: раздел `Closing a project` — два вызова агента, одна команда CLI, что блокеры НЕ разрешаются, неатомарность, известный лимит (агент может сам confirm).
- [x] 5.2 `go test ./... -count=1`, `gofmt`, `vet`, `validate --all --strict`
  **DoD:** всё green
  **Результат:** **156 тестов PASS** (было 142), `gofmt -l` пусто, `go vet` чист, `validate --all --strict` 11/11.
- [x] 5.3 **Negative DoD (стопор живой):** вызвать `close_project` без
  `confirm` на реальном демоне → проект остаётся `active`
  **DoD:** живой прогон: план возвращён, `GET /api/projects/{ref}` → `active`


## Границы

- Строгий lifecycle `todo → in_progress → review → done` и его валидация не
  трогаем.
- Точечный `done_step --force` не делаем.
- Персистентное состояние «ждёт подтверждения» не вводим.
- Известный остаток: агент с токеном может сам вызвать `confirm: true`. Это
  закрывает следующий ченж (токен подтверждения от человека).