# Proposal: fix-backup-ssh-path

## Why

Хост-факт (2026-10-06): пуш падает с `cannot run ssh: No such file or
directory`. Пакет `git` не поставляет бинарник `ssh` (он из `openssh`),
а remote на инстансе оказался SSH. Деградация снова отработала штатно:
данные коммитятся локально, пуш ретраится с warn.

## What Changes

- `path = [ pkgs.git pkgs.openssh ]` на уровне сервиса (не serviceConfig).
  Покрывает оба варианта remote (SSH и HTTPS) независимо от того, какой задан.
- Смена поведения remote: несовпавший origin переставляется на configured
  через `set-url` с громким warn вместо вечного отказа. Причина: отказ
  гарантировал залипший бэкап при каждой легитимной смене remote
  (ровно твой случай SSH -> HTTPS).

## Non-goals

- Не выясняем здесь почему remote SSH: это конфиг хоста, см. rollout.
  Код держит оба пути.

## Impact

- Affected specs: `store-git-backup` (MODIFIED: stale origin следует за
  конфигом вместо вечного отказа; имена сценариев сохранены).
- Affected code: `flake.nix` (path), `internal/gitbackup/gitbackup.go`
  (set-url), `internal/gitbackup/gitbackup_test.go` (mismatch-тест).
