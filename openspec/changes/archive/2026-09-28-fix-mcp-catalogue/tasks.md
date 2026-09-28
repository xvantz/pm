# Tasks: fix-mcp-catalogue

## Phase 1: Spec + test

- [x] 1.1 MODIFIED delta `Tool catalogue`: derived-формулировка, без чисел
  **DoD:** `openspec validate fix-mcp-catalogue --type change` green
- [x] 1.2 Тест уникальности имен в `TestToolsList_AllSchemasAreObjects`
  **DoD:** `CGO_ENABLED=0 go test ./internal/mcp/ -count=1` green

## Phase 2: Merge + archive

- [ ] 2.1 PR to main, CI green
  **DoD:** Forgejo run success
- [ ] 2.2 Archive, validate clean
  **DoD:** baseline каталог без чисел; `openspec list` без fix-mcp-catalogue
