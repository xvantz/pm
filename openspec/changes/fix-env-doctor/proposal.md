# Proposal: fix-env-doctor

## Why

Две операционные дыры (P8, P10 пула), premise проверен по коду 2026-10-01:

1. **Модуль не раздает PM_API.** `flake.nix` кладет `PM_TOKEN` в шеллы через
   sops-файл и `PM_DIR` через sessionVariables, а `PM_API` нет нигде.
   Клиенты падают на захардкоженный дефолт `apistore.DefaultAddr`
   (`internal/apistore/env.go`): смена `listenAddr` роняет всех молча.
   Курящий ствол найден в другом репо: `/dotfiles/.../hermes.nix:298`
   хардкодит `env.PM_API = "http://127.0.0.1:8472"` для MCP-сервера. Смена
   адреса роняет и шеллы, и агента одновременно, в двух репо сразу.
2. **Doctor молчит о расхождении.** `cmdDoctor` читает локальные файлы
   (`PM_DIR`), `doctorLive` ходит в демон (`PM_API`), но сравнить им нечего:
   демон не отдает свой dataDir нигде. Два `GET /healthz` поля (`status`,
   `version`), пути нет. Несовпадение `PM_DIR` vs пути демона сегодня
   невидимо по построению.

## What Changes

- **Flake модуль раздает PM_API** из `listenAddr` (`http://` + адрес): в
  `environment.sessionVariables` и в `interactiveShellInit` zsh/bash рядом с
  уже раздаваемым `PM_TOKEN`/`PM_DIR`. Не секрет - в nix store можно.
- **Dotfiles `hermes.nix` на тот же источник**: `env.PM_API` собирается из
  `config.services.pm.listenAddr` вместо хардкода. Модули живут в одном
  NixOS-конфиге (`hermes.nix` уже читает `config.services.pm.package`),
  так что это та же опция, а не дублирование значения.
- **Демон отдает свой dataDir**: `GET /healthz` добавляет поле `data_dir`.
  Путь не секрет, демон по умолчанию на localhost. `client.Health` декодит
  в `map[string]string` - толерантен, ничего не ломается.
- **Doctor сравнивает явно**: печатает file-сторону (`PM_DIR`) и daemon-сторону
  (`PM_API` + `data_dir` из healthz). Несовпадение корней - варнинг с обеими
  сторонами, а не молчание. Без демона сравнение пропускается (не ошибка).
- **README пример чинится**: `README.md:83,140` показывают захардкоженный
  адрес - обновить под derived.

## Non-goals

- Не переносим `PM_TOKEN` в sessionVariables: секрет, ему место только в
  sops-файле и шеллах. Разница с PM_API осознанная и записана здесь.
- Не заставляем MCP читать адрес из шелла: у контейнера Hermes нет login
  shell, ему адрес кладет модуль. Два пути раздачи (шеллы + контейнер) -
  не дублирование, а две разные среды.
- Не валидируем достижимость адреса в момент билда: демона может не быть
  во время `nixos-rebuild`, проверка - дело smoke, не сборки.

## Impact

- Affected specs: `serve-api` (MODIFIED: Nix options + env контракт - уже
  написанная дельта покрывает PM_API; ADDED: поле `data_dir` в healthz),
  `cli-lifecycle` (MODIFIED: Doctor - split с варнингом расхождения).
- Affected code: `flake.nix` (PM_API в sessionVariables + shellInit),
  `internal/api/server.go` (поле в healthz), `internal/client/client.go`
  (проброс поля наружу), `internal/cli/doctor_live.go` (сравнение и варнинг),
  `internal/cli/doctor*.go` тесты, README примеры.
  Кросс-репо: `/dotfiles/modules/system/hermes/hermes.nix` (`env.PM_API` из
  `services.pm.listenAddr`) - отдельным коммитом туда, не сюда.
- Соседи (проверены грепом, правило 4):
  - `hermes-integration:11` документирует `env.PM_API` с дефолтом - значение
    по умолчанию не меняется, правок не надо.
  - `serve-api` сценарий health/Auth (`200 {"status":"ok","version":"..."}`):
    поле добавляется, старые клиенты (`map[string]string`) не ломаются.
    Сценарий расширяется в этой же дельте.
  - `fix-event-time` граница (legacy-счетчик, зона дня) - не трогаем, в
    proposal того ченжа уже записана обратная граница.
- Пересечения по файлам: активных нет. `flake.nix`, `doctor*.go` ни один
  открытый ченж не трогает.
- P8/P10 пула закрываются архивом этого ченжа.
