# Proposal: add-backup-token-auth

## Why

Раскатка бэкапа уперлась в auth: deploy-ключи под репо это запарка
(генерация, регистрация, write-флаг), а персональный токен уже есть в
рабочем флоу Ивана. Код при этом умеет только SSH-ключ: `tokenFile`
некуда положить, HTTPS-remote нечем авторизовать.

## What Changes

- `gitbackup.Config` получает `Token`: когда задан, каждый git-процесс
  несет `Authorization: Bearer` через env-конфиг (`GIT_CONFIG_COUNT` /
  `GIT_CONFIG_KEY_0=http.extraHeader` / `GIT_CONFIG_VALUE_0`), а не через
  `-c` в командной строке (токен не светится в `ps`). `KeyFile` остается
  для SSH; заданы оба - каждый едет своим каналом, git берет подходящий
  под схему remote.
- `pm serve` получает `--backup-token` + `PM_BACKUP_TOKEN` env.
- Nix: `services.pm.backup.tokenFile` (sops-секрет, та же конвенция что
  `pm_env`), проброс в сервис как `PM_BACKUP_TOKEN` только когда задан.
- Тесты: env несет заголовок при заданном токене и не несет без него;
  коммит+пуш в bare с токеном (file-remote, заголовок игнорируется).
  Честное ограничение: E2E по HTTPS локально не поднять без http-git
  сервера - покрытие на уровне env + file-remote.

## Non-goals

- Не выпиливаем SSH-путь: ключ остается рабочей опцией.
- Не кладем токен в URL remote (`https://token@host` светит в логах
  и конфиге) и не пишем его на диск (никаких `.netrc`).
- Не расширяем скоупы: какой токен дать - решение раскатки (repo write),
  код ест любой Bearer.

## Impact

- Affected specs: `store-git-backup` (MODIFIED: опции несут repo, key
  и token).
- Affected code: `internal/gitbackup/gitbackup.go` (env заголовка),
  `internal/cli/serve.go` (флаг/env), `flake.nix`
  (`services.pm.backup.tokenFile` + проброс), тесты.
