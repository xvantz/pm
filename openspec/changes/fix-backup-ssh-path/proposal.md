# Proposal: fix-backup-ssh-path

## Why

Хост-факт (2026-10-06): пуш падает с `cannot run ssh: No such file or
directory`. Пакет `git` не поставляет бинарник `ssh` (он из `openssh`),
а remote на инстансе оказался SSH. Деградация снова отработала штатно:
данные коммитятся локально, пуш ретраится с warn.

## What Changes

- `path = [ pkgs.git pkgs.openssh ]` на уровне сервиса. Покрывает оба
  варианта remote (SSH и HTTPS) независимо от того, какой задан.

## Non-goals

- Не выясняем здесь почему remote SSH: это конфиг хоста, см. rollout.
  Код держит оба пути.

## Impact

- Affected specs: нет. Дельты нет (`skip_specs`).
- Affected code: `flake.nix` (один пакет в path).
