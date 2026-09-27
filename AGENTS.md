# AGENTS.md — PM repo agent rules

Trigger phrase: `ведем по openspec`. When active (or always in this repo):

1. OpenSpec cycle is mandatory: propose → approval → apply → archive. No code before proposal approval.
2. Every task carries `**DoD:**` (a command or observation, never prose). `[x]` means DoD green on the live project.
3. Propose only from a clean `git status` tree. One change fully closed before the next opens.
4. `openspec validate --all --strict` zero-fail before every commit.
5. Push to branches + PR. Never direct to main.
6. Close definition: change gone from `openspec list` (archived) AND baseline carries the delta AND CI green. 6/9 is not done.

Hooks: `git config core.hooksPath scripts/hooks` (pre-commit: gofmt + validate; pre-push: vet + test, CGO_ENABLED=0).
CI: `.forgejo/workflows/ci.yml` (checkout@v7, setup-go@v7, validate strict, build, vet, test).
