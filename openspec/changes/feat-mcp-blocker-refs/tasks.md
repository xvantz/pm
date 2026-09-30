# Tasks: feat-mcp-blocker-refs

## Phase 1: Код

- [x] 1.1 `list_steps`: поле `blocker_ids` в кратком шаге
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -run TestBoundedReads_ListStepsOmitsHeavyFields -count=1` зелен и дополнен проверкой id
  **Результат:** `jsonStepBrief.blocker_ids` всегда в ответе, причины не едут. Новый `TestBlockerRefs_ListStepsNamesBlockers` проверяет `configure-dns -> [router]` и пустой список у свободного шага.
- [x] 1.2 `list_blockers`: опциональный `step_id` фильтр
  **DoD:** вызов со `step_id` возвращает только группу этого шага; без `step_id` отвечает как раньше
  **Результат:** фильтр + ошибка с именем шага для неизвестного `step_id`. Проверено `TestBlockerRefs_ListBlockersFiltersByStep` и `TestBlockerRefs_ListBlockersUnknownStepFails`.
- [x] 1.3 Описания тулов: `list_steps` упоминает `blocker_ids`, `list_blockers` упоминает `step_id`
  **DoD:** `tools/list` содержит новые описания, `validate --all --strict` зелен
  **Результат:** описания обновлены, каталог derived, валидация ниже зеленая.

## Phase 2: Доказательства

- [x] 2.1 Тесты на оба пути: id в списке ведут в `get_step`/`resolve_blocker`
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1` зелен
  **Результат:** `TestBlockerRefs_IdsLeadToGetStep` гоняет каждую непустую связку список -> детальный шаг.
- [x] 2.2 Полный прогон: build, vet, fmt, validate strict
  **DoD:** `CGO_ENABLED=0 go test ./... -count=1` зелен; `validate --all --strict` показывает 0 failed
  **Результат:** все пакеты зелены, vet и gofmt чисты, строгая валидация ниже зеленая.

## Phase 3: Финал

- [x] 3.1 README: строка про `blocker_ids` и фильтр `step_id`
  **DoD:** раздел Reading through MCP упоминает оба
  **Результат:** таблица тулов и раздел чтения обновлены.
- [ ] 3.2 PR + CI + архив последним коммитом
  **DoD:** Forgejo run success; `openspec list` без `feat-mcp-blocker-refs`

## Границы

- Нового тула `get_blocker` нет. Отклонен сознательно: каталог не пухнет ради экономии 200 B.
- HTTP API демона не меняем.
- `reason` остается только в `get_step` и `list_blockers`, не в `list_steps`.
