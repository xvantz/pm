# Tasks: feat-close-consent → feat-close-consent (переименован, DoD обязателен)

Правило 2 нарушено всеми 5 открытыми ченджами — здесь DoD есть на каждой задаче.
Чендж переписан по итогам обсуждения: вместо `done_step --force` (решает
несуществующую боль, ослабляет стопор) — `close_project` требует согласия.

## Phase 1: Store (ClosePlan)

- [ ] 1.1 `internal/types`: `ClosePlan` — список шагов к закрытию (ID, title,
  статус) + висящие блокеры (ID, title, reason, step_id)
  **DoD:** `CGO_ENABLED=0 go build ./...` green; тип экспортирован из `types`
- [ ] 1.2 `internal/store/store.go`: `ClosePlan(projectID) (*ClosePlan, error)`
  в интерфейсе
  **DoD:** `go build ./...` green; все реализации `Store` удовлетворяют
  интерфейсу (`FileStore`, `MockStore`)
- [ ] 1.3 `internal/store/filestore.go` + `mock.go`: реализация `ClosePlan` —
  read-only, ничего не мутирует
  **DoD:** тест: `ClosePlan` на проекте с 3 открытыми степ-ами и 1 блокером
  возвращает 3 шага с их статусами и 1 блокер; после вызова `GetProject`
  возвращает идентичные данные (не изменилось ничего)
- [ ] 1.4 Юнит-тест: `ClosePlan` на `completed` проекте → пустой план, не ошибка
  **DoD:** `CGO_ENABLED=0 go test ./internal/store/ -count=1` green

## Phase 2: API (consent на эндпоинте)

- [ ] 2.1 `internal/api/server.go` `handleProjectClose`: поле `Confirm bool`;
  без `confirm` → `ClosePlan` в ответе, проект не тронут; с `confirm` и пустым
  reason → 4xx, не тронут
  **DoD:** тест `internal/api/server_test.go`: POST `/close` без `confirm` →
  200 с планом, `GET /api/projects/{ref}` после показывает `active` и
  исходные статусы; POST с `confirm:true` без reason → 422, проект `active`;
  POST с `confirm:true` и reason → 200, `completed`
- [ ] 2.2 Клиент `internal/client/client.go`: `ClosePlan` + `Confirm` в `CloseProject`
  **DoD:** `go build ./...` green; сигнатура покрывает оба вызова
- [ ] 2.3 `internal/apistore/apistore.go`: проброс `ClosePlan`
  **DoD:** `CGO_ENABLED=0 go build ./...` green

## Phase 3: MCP

- [ ] 3.1 `internal/mcp/tools.go` `close_project`: `confirm` в inputSchema;
  без `confirm` — текстовый план (список шагов + блокеры + предупреждение),
  с `confirm` и reason — закрытие
  **DoD:** MCP-тест: вызов без `confirm` возвращает план, `get_project` после
  показывает `active`
- [ ] 3.2 MCP-тесты: confirm без reason падает; confirm с reason закрывает;
  completed → already completed
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1` green

## Phase 4: CLI

- [ ] 4.1 `internal/cli/project.go`: `pm project close <id> <reason>` — reason
  обязателен (пустой → ошибка, дефолт `bulk close` из CLI убран)
  **DoD:** тест: `cmdProjectClose(["1"])` возвращает ошибку; с reason закрывает
- [ ] 4.2 Help-текст: одна команда, reason обязателен
  **DoD:** `pm help` содержит обновлённую строку close

## Phase 5: Проверка

- [ ] 5.1 `README.md`: раздел про закрытие — план, согласие, reason; почему
  CLI в одну команду, а агент в два
  **DoD:** раздел есть, пример обоих вызовов
- [ ] 5.2 `go test ./... -count=1`, `gofmt`, `vet`, `validate --all --strict`
  **DoD:** всё green
- [ ] 5.3 **Negative DoD (стопор живой):** вызвать `close_project` без
  `confirm` на реальном демоне → проект остаётся `active`
  **DoD:** живой прогон: план возвращён, `GET /api/projects/{ref}` → `active`

## Границы

- Строгий lifecycle `todo → in_progress → review → done` и его валидация не
  трогаем.
- Точечный `done_step --force` не делаем.
- Персистентное состояние «ждёт подтверждения» не вводим.
- Известный остаток: агент с токеном может сам вызвать `confirm: true`. Это
  закрывает следующий ченж (токен подтверждения от человека).
