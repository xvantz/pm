# Tasks: chore-mcp-actionable-responses

## Phase 1: Ответы на языке тулов

- [ ] 1.1 `Hint` сводки + все `Next:` пишущих тулов: CLI-команды в имена тулов
  `add_project`, `add_step`, `start/review/done_step`, `add/resolve_blocker`,
  `add_decision`, `close_project`, `delete_project`, `trash_restore`
  **DoD:** `grep -n 'Next: pm \|pm list_\|pm step\|pm blocker\|pm project\|pm add' internal/mcp/tools.go` пуст;
  каждый хинт называет существующий тул
  **Файлы:** `internal/mcp/tools.go` (только строки ответов)

- [ ] 1.2 `get_step`: добавить `hint`
  blocked - куда за причиной и чем снять; иначе следующий по статусу
  **DoD:** тест: blocked-шаг несет `resolve_blocker`, обычный - следующий шаг
  жизненного цикла
  **Файлы:** `internal/mcp/tools.go` (ответ get_step)

## Phase 2: Actionable ошибки + финал

- [ ] 2.1 Обертки ошибок в `handle*`
  unknown step, already exists x3, missing/invalid args, confirm без reason,
  done без review (поднять доменный + чинящий вызов)
  **DoD:** каждая покрытая ошибка содержит действие; негатив: старый текст
  роняет тест
  **Файлы:** `internal/mcp/tools.go` (только тексты ошибок),
  `internal/mcp/actionable_responses_test.go`

- [ ] 2.2 Полный прогон + PR + архив
  build, vet, fmt, `validate --all --strict`, Forgejo run success
  **DoD:** run success; `openspec list` без ченжа; baseline несет дельту

## Границы

- Доменные тексты не трогаем (обертка на handle-уровне).
- Старт кода после мержа #32 (тот же файл).
