# pm — Project Memory

Система долговременной памяти проектов для человека и ИИ.

Хранит не список задач, а **текущее состояние проектов, принятые решения, блокеры, историю изменений и следующие шаги**.

Главная цель — сократить время восстановления контекста проекта с десятков минут до менее чем 30 секунд.

## Концепция

Информация о проектах распределена между Git-репозиториями, Obsidian, чатами с ИИ и памятью человека.
PM становится единым источником истины о состоянии проекта.

## Установка

```bash
go install github.com/xvantz/pm/cmd/pm@latest
```

Или собери из исходников:

```bash
git clone <url>
cd pm
go build -o pm ./cmd/pm
sudo mv pm /usr/local/bin/
```

## Использование

### Инициализация

```bash
cd ~/projects
pm init
```

Создаёт `./pm/` с git-репозиторием и структурой `projects/`. Внутри:

```
pm/
├── .git/
├── .gitignore
└── projects/     ← здесь живут YAML-файлы проектов
```

### Брифинг

```bash
cd ~/projects
pm briefing              # брифинг из ./pm/projects/
pm briefing --mock       # тестовый брифинг с демо-данными (везде, без --dir)
pm briefing --json       # вывод в JSON для LLM-агента
pm briefing --date 2026-06-13  # брифинг на конкретную дату
pm briefing --dir /path/to/projects  # кастомный путь
```

### Структура проектов

```
~/Documents/pm/projects/
├── agh/
│   ├── project.yaml       # метаданные проекта
│   ├── steps/             # шаги (каждый — отдельный .yaml)
│   ├── blockers/          # блокеры
│   └── decisions/         # принятые решения
├── pm/
└── ...
```

## Формат данных

### Project

```yaml
id: agh
title: AdGuard Home
goal: "Развернуть домашний DNS сервер"
status: active
tags:
  - infrastructure
  - homelab
created_at: "2026-06-10"
updated_at: "2026-06-14"
completed_at:              # только если status: completed
```

### Step

```yaml
id: setup-caddy
title: "Настроить Caddy reverse proxy"
status: done               # todo | in_progress | review | done | blocked
project_id: agh
artifacts:
  - docs/caddy-setup.md    # ссылки на PR, коммиты, файлы
deps:                      # опционально
  - install-agh
```

### Blocker

```yaml
id: router
title: "Купить GL.iNet роутер"
reason: "Нет свободного бюджета"
status: waiting
project_id: agh
```

### Decision

```yaml
id: migrate-docker
title: "Отказ от Docker в пользу NixOS Service"
reason: "NixOS Service проще сопровождать"
date: "2026-06-13"
project_id: agh
```

## Тестирование

```bash
go test ./...
```

Или с детальным выводом:

```bash
go test -v ./...
```

Тесты покрывают:
- генерацию брифинга (базовая, сводка, Markdown, JSON, блокеры, даты, пустое хранилище, приоритеты рекомендаций)
- store (мок: CRUD, поиск; файловый: создание/чтение/запись проектов, шагов, блокеров, пустая директория)
- типы (статусы, артефакты, теги, композиция ProjectData)

## Команды

- `pm briefing` — ✅
- `pm init` — ✅
- `pm project create <title>` — ✅ авто-ID из названия, статус `idea`
- `pm project list` — ✅
- `pm project show <id>` — ✅ шаги, блокеры, решения
- `pm project goal <id> <text>` — ✅
- `pm project tag <id> <tag>...` — ✅
- `pm step add <id> <title>` — ✅ ID из названия (slug)
- `pm step done <id> <step-id>` — ✅
- `pm step list <id>` — ✅
- `pm blocker add <id> <title>` — ✅
- `pm blocker resolve <id> <blocker>` — ✅
- `pm blocker list <id>` — ✅
- `pm decision add <id> <title>` — ✅
- `pm decision list <id>` — ✅
- MCP-обёртка — пул-реквестом после стабилизации CLI

## Лицензия

MIT
