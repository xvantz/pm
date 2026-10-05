# Proposal: fix-backup-git-path

## Why

Прод-деградация с живым логом (2026-10-06, хост): демон стартовал с
включенным бэкапом, но упал в `serving without backup` с
`git init -b main: exec: "git": executable file not found in $PATH`.
systemd-сервис живет с минимальным PATH, `git` туда не входит.
Хорошая новость: контракт "деградация вместо отказа" сработал -
API служит, бэкап громко лежит в логе.

## What Changes

- `path = [ pkgs.git ]` в `serviceConfig` pm-serve: гит в PATH демона.
  Один параметр, без смены логики.

## Non-goals

- Не меняем поведение без гита: деградация остается как есть.
- Не тащим полный `gitFull`: хватает минимального `pkgs.git`.

## Impact

- Affected specs: нет (поведение спеки не меняется, чинится окружение).
  Дельты нет, в ченже `skip_specs`.
- Affected code: `flake.nix` (одна строка).
