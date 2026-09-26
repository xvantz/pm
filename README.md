# PM — Project Memory

Долговременная память проектов для человека и ИИ.

Хранит состояние проектов, шаги, блокеры, архитектурные решения — в YAML-файлах под Git. CLI для человека + MCP для агентов.

---

## Концепция

PM решает проблему восстановления контекста: вместо того чтобы перечитывать чаты, Obsidian и git-логи, открываешь `pm briefing` — и за 30 секунд понимаешь, что происходит с проектами, что изменилось, что блокирует, что делать today.

## Архитектура

```
cmd/              # entry points
├── pm/           # CLI (человек)
└── pm-mcp/       # MCP-сервер (агенты, поверх stdio)
internal/
├── cli/          # CLI-хендлеры
├── mcp/          # JSON-RPC 2.0, Content-Length framing
├── domain/       # бизнес-логика (шаговая машина)
├── store/        # FileStore + MockStore + interface
├── slug/         # slug-генератор
├── types/        # модели данных
└── briefing/     # движок ежедневных сводок
```

## Установка

### Nix Flake

```nix
# flake.nix
{
  inputs.pm.url = "github:xvantz/pm";

  outputs = { pm, ... }: {
    nixosConfigurations.nixos = nixpkgs.lib.nixosSystem {
      modules = [
        pm.nixosModules.default
        { services.pm.enable = true; }
      ];
    };
  };
}
```

Билд:

```bash
nix build github:xvantz/pm
# или
nix run github:xvantz/pm#pm -- project list
```

### Go (dev)

```bash
git clone https://git.827482.xyz/xvantz/pm ~/Documents/pm
cd ~/Documents/pm
go build -o ~/.local/bin/pm ./cmd/pm
go build -o ~/.local/bin/pm-mcp ./cmd/pm-mcp
```

## Использование

```bash
export PM_DIR=~/Documents/pm/projects

pm init                                          # инициализировать хранилище
pm project create "AdGuard Home"                 # создать проект
pm add step #1 "Настроить Caddy"                 # добавить шаг
pm step start #1 setup-caddy                    # начать шаг
pm step review #1 setup-caddy                   # отправить на ревью
pm step done #1 setup-caddy                     # завершить (только из review)
pm blocker add --reason "нет бюджета" #1 setup-caddy "Купить роутер"
pm blocker resolve #1 setup-caddy router         # снять блокер
pm decision add --reason "один бинарник" #1 "Go как язык"
pm doctor                                        # проверка целостности
pm trash list                                    # проекты в корзине
pm trash restore <name>                          # восстановить
pm briefing                                      # ежедневная сводка
```

### MCP (для агентов)

PM-mcp — JSON-RPC 2.0 сервер поверх stdio с Content-Length фреймингом. 14 инструментов:

| Инструмент | Описание |
|-----------|----------|
| `list_projects` | список проектов (JSON) |
| `get_project` | проект + шаги + решения (JSON) |
| `add_project` | создать проект |
| `add_step` | добавить шаг |
| `start_step` | todo → in_progress |
| `review_step` | отправить на ревью |
| `done_step` | завершить (только из review) |
| `add_blocker` | добавить блокер |
| `resolve_blocker` | снять блокер |
| `add_decision` | записать решение |
| `get_briefing` | сгенерировать сводку |
| `list_steps` | шаги проекта (JSON) |
| `list_blockers` | блокеры (JSON) |
| `list_decisions` | решения (JSON) |

### Интеграция с Hermes Agent

```nix
# hermes.nix
{ config, ... }: {
  services.hermes-agent.mcpServers.pm = {
    command = "${config.services.pm.package}/bin/pm-mcp";
    env.PM_DIR = config.services.pm.dataDir;
  };
}
```

После `nixos-rebuild` все 14 инструментов доступны с префиксом `mcp_pm_`.

## Модель данных

```yaml
# <project-id>/project.yaml
id: "0196f1a2-..."
number: 1
title: "AdGuard Home"
goal: "Развернуть DNS-сервер"
status: active          # idea | active | paused | completed
tags: ["infrastructure"]
created_at: "2026-06-10"
updated_at: "2026-06-14"
```

```yaml
# <project-id>/steps/<slug>.yaml
id: setup-caddy
title: "Настроить Caddy"
status: done            # todo | in_progress | review | done | blocked
project_id: "0196f1a2-..."
blockers: []
created_at: "2026-06-13"
updated_at: "2026-06-13"
```

```yaml
# blockers живут внутри step.yaml:
blockers:
  - id: router
    title: "Купить роутер"
    reason: "Нет бюджета"
    status: waiting      # waiting | active | resolved
    project_id: "..."
    step_id: "configure-dns"
    created_at: "2026-06-10"
    updated_at: "2026-06-10"
```

```yaml
# <project-id>/decisions/<slug>.yaml
id: use-go
title: "Go как язык"
reason: "Один бинарник, без зависимостей"
date: "2026-06-13"
project_id: "..."
```

## Надёжность

| Свойство | Механизм |
|----------|----------|
| Атомарность записи | temp → sync → rename |
| Durability | fsync файла + родительской директории |
| Cross-process locking | flock на `.pm.lock` |
| Graceful shutdown | signal.NotifyContext |
| Recovery | корзина (.trash), NextNumber fallback scan |
| Диагностика | `pm doctor` проверяет целостность |

## Тесты

```bash
go test ./... -count=1   # 112+ тестов, 7 пакетов
go vet ./...
```

## Демон (`pm serve`)

Один писатель на хранилище: демон владеет YAML, клиенты ходят по HTTP.
Контейнерам не нужны маунты данных, только адрес и токен.

```bash
export PM_TOKEN=$(openssl rand -hex 32)
pm serve --addr 127.0.0.1:8472 &
curl localhost:8472/healthz
curl -H "Authorization: Bearer $PM_TOKEN" localhost:8472/api/projects
```

| Метод | Путь | Что делает |
|-------|------|-----------|
| GET | /healthz | liveness, без auth |
| GET/POST | /api/projects | список / создать `{title, goal?, tags?}` |
| GET/PATCH/DELETE | /api/projects/{ref} | детали / `{goal?, status?, tags?}` / в корзину |
| GET/POST | /api/projects/{ref}/steps | список / создать `{title}` |
| POST | .../steps/{step}/{start,review,done} | lifecycle с валидацией |
| DELETE | .../steps/{step} | удалить шаг |
| GET | /api/projects/{ref}/blockers | список блокеров |
| POST | .../steps/{step}/blockers | создать `{title, reason?}` |
| POST | .../blockers/{blk}/resolve | снять блокер |
| DELETE | .../blockers/{blk} | удалить блокер |
| GET/POST | /api/projects/{ref}/decisions | список / создать `{title, reason?}` |
| DELETE | .../decisions/{dec} | удалить решение |
| GET | /api/briefing?date=&project= | брифинг JSON |

Коды: 400 тело, 401 auth, 404 ref, 409 дубликат, 422 lifecycle, 500 store.
Слушает localhost по умолчанию. Наружу только через Tailscale, никогда 0.0.0.0.

## Клиенты без файлов (remote mode)

CLI и MCP умеют работать через демон вместо YAML. Нужны только адрес и токен,
маунты данных не нужны:

```bash
export PM_API=http://127.0.0.1:8472 PM_TOKEN=...
pm project list     # идет в демон, тексты 1:1 как локально
pm briefing         # брифинг считает демон
```

`pm-mcp` в remote режиме не проверяет папку вообще. `pm doctor` всегда
локальный (целостность файлов через API не проверяют). `pm serve` всегда
пишет напрямую в файлы.

## Лицензия

MIT
