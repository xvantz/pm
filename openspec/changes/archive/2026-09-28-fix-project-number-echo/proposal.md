# Proposal: fix-project-number-echo

## Why

`pm project create` печатает advisory номер (локальный NextNumber), а сервер назначает настоящий под мьютексом. При гонке подсказки ведут не туда (P5 пула).

## What Changes

- CLI после SaveProject перечитывает проект по ID и печатает серверный номер.
- ADDED требование в cli-lifecycle: create echo всегда серверный номер.

## Impact

- Affected specs: `cli-lifecycle` (ADDED: 1 requirement).
- Affected code: `internal/cli/project.go` (re-read после save).
- MCP путь уже возвращает созданное с серверным номером, правок не надо.
