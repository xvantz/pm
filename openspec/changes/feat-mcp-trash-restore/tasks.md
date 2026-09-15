# Tasks: feat-mcp-trash-restore

## Phase 1: Store

- [ ] 1.1 `FileStore`: бэкап YAML в `_meta/backups/<timestamp>/` перед удалением, ротация последних 20
- [ ] 1.2 Единый `Restore` в store, используемый CLI и MCP; неоднозначное имя - ошибка со списком кандидатов
- [ ] 1.3 Юнит-тесты: удаление пишет бэкап, restore возвращает, неоднозначность падает

## Phase 2: MCP surface

- [ ] 2.1 Тулы `trash_list` + `trash_restore` в `RegisterPMTools` со схемами `type: object` (см. fix-mcp-list-projects-schema)
- [ ] 2.2 MCP-тесты обоих тулов
- [ ] 2.3 README: раздел про корзину и бэкапы
