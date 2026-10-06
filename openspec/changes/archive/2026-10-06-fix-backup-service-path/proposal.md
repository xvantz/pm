# Proposal: fix-backup-service-path

## Why

Хост-факт (2026-10-06): после свитча с предыдущим фиксом юнит говорит
`Unknown key 'path' in section [Service], ignoring` - `path` лежит в
`serviceConfig`, а это сквозной проброс в systemd, который такого ключа
не знает. Правильное место - атрибут сервиса (`systemd.services.<name>.path`,
штатная опция NixOS, кладет пакеты в PATH юнита).

## What Changes

- `path = [ pkgs.git ]` переезжает из `serviceConfig` на уровень сервиса
  с комментарием-предупреждением (именно эта ошибка уже случалась).
- Неизвестная опция на уровне сервиса роняет evaluation громко, а не
  игнорит тихо - худший исход это ошибка сборки, не молчаливый слом.

## Non-goals

- Поведение не меняется, только место опции.

## Impact

- Affected specs: нет. Дельты нет (`skip_specs`).
- Affected code: `flake.nix` (перемещение строк).
