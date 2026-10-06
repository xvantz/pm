# Proposal: fix-backup-sterile-git-env

## Why

Хост-факт (2026-10-06): origin уже https, а пуш уходит по SSH с
`Permission denied (publickey)`. Единственный механизм: пользовательский
`url.insteadOf` в git-конфиге хоста тихо переписывает транспорт.
Демон обязан не зависеть от пользовательских git-настроек.

## What Changes

- Все git-процессы бэкапа бегут в стерильном конфиге:
  `GIT_CONFIG_NOSYSTEM=1` + `GIT_CONFIG_GLOBAL=/dev/null`. Действует только
  repo-local конфиг (identity там запинена). Проверено локально: при
  дублирующихся env git берет последнее значение, /dev/null читается как
  пустой конфиг.
- Регресс-тест: враждебный global insteadOf не перехватывает транспорт.

## Non-goals

- Не трогаем пользовательский конфиг хоста: insteadOf там нужен
  интерактивной работе, демон просто его не видит.
- Поведение коммитов/пуша не меняется.

## Impact

- Affected specs: `store-git-backup` (MODIFIED: +сценарий про игнор
  пользовательских переписок транспорта; имена сценариев сохранены).
- Affected code: `internal/gitbackup/gitbackup.go` (env),
  `internal/gitbackup/gitbackup_test.go`.
