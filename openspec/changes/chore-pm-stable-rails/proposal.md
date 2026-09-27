# Proposal: chore-pm-stable-rails

## Why

PM репо живет без enforcement пачки: нет CI, нет хуков, нет AGENTS.md клаузы. Правила openspec держатся на памяти агента, а память не gate. Первый тест скилла openspec-enforcement: поставить PM на стабильные рельсы и доказать gates негативными проверками.

## What Changes

- `.forgejo/workflows/ci.yml`: `validate --all --strict` + `go build/vet/test` (CGO_ENABLED=0, контейнер без gcc). Merge в main только зеленым.
- Git hooks: pre-commit (gofmt + validate), pre-push (vet + test). lefthook бинаря в env нет, поэтому tracked скрипты в `scripts/hooks/` + install в `.git/hooks/`.
- `AGENTS.md`: openspec цикл обязателен, DoD на каждом таске, propose только с чистого дерева, один ченж за раз.
- PM трекер: шаг под этот ченж в проекте 8.
- PR под уже запушенную ветку `chore/archive-done-specs` (открытый хвост).

## Impact

- Affected code: repo root (CI, hooks, AGENTS.md). Product код не трогается.
- Spec delta: нет (tooling change, продуктовая спека не меняется).
- Out of scope: `feat/pm-close` merge, trash-restore, lifecycle-force. Они следующие в очереди, не в этом ченже.
