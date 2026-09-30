# AGENTS.md — PM repo agent rules

Trigger phrase: `ведем по openspec`. When active (or always in this repo):

1. OpenSpec cycle is mandatory: propose → approval → apply → archive. No code before proposal approval.
2. Every task carries `**DoD:**` (a command or observation, never prose). `[x]` means DoD green on the live project.
3. Several changes may stay open at once. The gate is **file overlap, not order**: two open changes that both edit the same file are not merged independently — the second waits, or moves into its own `git worktree` (`git worktree add ../pm-<id> -b <type>/<id>`). `Affected code` in each proposal is the declaration of overlap; if two proposals list the same file, say so in both and pick the merge order.
4. Before merging a change that touches an existing requirement, check the neighbours: grep the baseline spec for the change's own nouns and markers (exactly, only, never, always, counts). Every affected statement rides in the same delta as MODIFIED, or the proposal says why it does not. A MODIFIED that quietly drops a scenario is drift.
5. `openspec validate --all --strict` zero-fail before every commit.
6. Push to branches + PR. Never direct to main.
7. Close definition: change gone from `openspec list` (archived) AND baseline carries the delta AND CI green. 6/9 is not done. Archive as the last commit of the same PR, never a follow-up PR.

Hooks: `git config core.hooksPath scripts/hooks` (pre-commit: gofmt + validate; pre-push: vet + test, CGO_ENABLED=0).
CI: `.forgejo/workflows/ci.yml` (checkout@v7, setup-go@v7, validate strict, build, vet, test).
