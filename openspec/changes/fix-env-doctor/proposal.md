# Proposal: fix-env-doctor

## Why

Две операционные дыры (P8, P10 пула): модуль не раздает PM_API (смена listenAddr роняет клиентов, сейчас только дефолт спасает), doctor проверяет локальный PM_DIR в то время как демон может смотреть в другое место. Remote-only наполовину: адрес захардкожен конвенцией.

## What Changes

- Flake модуль раздает PM_API (адрес из listenAddr) в шеллы и MCP env рядом с PM_TOKEN.
- Doctor: явное разделение file-целостность (PM_DIR) vs daemon-доступ (PM_API), несовпадение путей подсвечивается а не молчит.
- AGENTS.md/док обновить под оба env.

## Impact

- Affected specs: `serve-api` (MODIFIED: Nix options + env контракт), `cli-lifecycle` (MODIFIED: Doctor).
- Affected code: `flake.nix`, `internal/cli/doctor*.go`, dotfiles `hermes.nix` (кросс-репо часть).
