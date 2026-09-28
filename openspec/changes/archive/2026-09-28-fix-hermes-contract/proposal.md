# Proposal: fix-hermes-contract

## Why

Launch contract в hermes-integration спеке описывает несуществующий мир: `--dir`, `containerDataDir`, bind-mount. Код remote-only со времен serve-daemon (P2 пула). Док hermes-integration.md при этом правдив, но врет в счетчике тулов (13 при 16).

## What Changes

- MODIFIED `Launch contract`: запуск без --dir и маунтов, контракт "адрес + токен", PM_DIR только хост-путь демона. Сценарий старта переписан под remote-only, Clean build сохранен.
- Док: счетчик тулов 13 → 16 с полным списком.

## Impact

- Affected specs: `hermes-integration` (MODIFIED: 1 requirement).
- Affected docs: `hermes-integration.md` (1 строка + список).
- Кода нет, один коммит. Соседи проверены грепом: containerDataDir остается только в архиве.
