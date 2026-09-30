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

**Граница с fix-event-time:** счётчик legacy/битых меток в `pm doctor` и правило
зоны дня уже реализованы и специфицированы в `fix-event-time`
(`project-store`: Event timestamps, Calendar-day bucketing). Этот ченж НЕ трогает
эти аспекты: его `MODIFIED: Doctor` касается только разделения file-vs-daemon
проверок и вывода про PM_API/PM_DIR. Не дублировать.
