# Tasks: fix-hermes-contract

- [x] 1.1 MODIFIED delta `Launch contract` под remote-only реальность
  **DoD:** `openspec validate fix-hermes-contract --type change` green
- [x] 1.2 Док: счетчик тулов 13 → 16
  **DoD:** список в доке один в один с tools/list (16 штук)
- [ ] 2.1 PR to main, CI green
  **DoD:** Forgejo run success
- [ ] 2.2 Archive, validate clean
  **DoD:** baseline без containerDataDir вне архива; `openspec list` без fix-hermes-contract
