# Tasks: chore-pm-stable-rails

## Phase 1: Pack

- [ ] 1.1 CI workflow `.forgejo/workflows/ci.yml` (actions/checkout@v7, actions/setup-go@v7, validate strict + build/vet/test, CGO_ENABLED=0)
  **DoD:** Forgejo run зеленым на ветке ченжа
- [x] 1.2 Git hooks: tracked `scripts/hooks/pre-commit` (gofmt + validate) и `pre-push` (vet + test), install в `.git/hooks/`
  **DoD:** негативная проверка локально: битый gofmt реджектит pre-commit, фикс проходит — DONE both directions (red on 9 unformatted files, green after gofmt -w + validate 8/8 + vet/test)
- [x] 1.3 `AGENTS.md` клауза: цикл обязателен, DoD на таске, propose с чистого дерева, один ченж за раз, main только через PR
  **DoD:** файл в корне, содержит trigger-фразу и close definition — DONE
- [x] 1.4 PM трекер: шаг `chore-pm-stable-rails` в проекте 8
  **DoD:** step виден в проекте через MCP — DONE (in_progress, project 8)

## Phase 2: Prove + close

- [ ] 2.1 Push ветки, CI зеленым
  **DoD:** runs API показывает status success на sha ветки
- [ ] 2.2 Archive ченжа, validate чистым
  **DoD:** `openspec list` показывает только trash-restore, lifecycle-force, feat-pm-close; `validate --all --strict` без новых ошибок
- [ ] 2.3 PR под `chore/archive-done-specs` (хвост прошлого шага)
  **DoD:** PR open в Forgejo xvantz/pm
