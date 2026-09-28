# Tasks: fix-project-number-echo

- [x] 1.1 ADDED delta: create echo серверного номера
  **DoD:** `openspec validate fix-project-number-echo --type change` green
- [x] 1.2 Код: re-read после SaveProject в `cmdProjectCreate`
  **DoD:** `CGO_ENABLED=0 go test ./internal/cli/ -count=1` green + параллельный create дает верные номера в подсказках
- [ ] 2.1 PR to main, CI green
  **DoD:** Forgejo run success
- [ ] 2.2 Archive, validate clean
  **DoD:** `openspec list` без fix-project-number-echo
