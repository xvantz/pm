# PM: Оставшиеся системные проблемы — план фиксов

> **Для Hermes:** Использовать `subagent-driven-development` для выполнения по задачам.

**Цель:** Закрыть все выявленные в senior-ревью проблемы — от quick wins до архитектурного DRY.

**Архитектура:** Четыре фазы, каждая может быть остановлена после неё. Фазы независимы.

---

## Фаза 1: Quick wins (очередь: 1→2→3→4)

### Task 1: `loadProjectData` — не глотать ошибки шагов/решений

**Проблема:** `filestore.go:281-295`:
```go
func (s *FileStore) loadProjectData(p types.Project) (*types.ProjectData, error) {
    steps, err := s.GetSteps(p.ID)
    if err != nil {
        steps = nil  // ← ошибка проглочена
    }
    decisions, err := s.GetDecisions(p.ID)
    if err != nil {
        decisions = nil  // ← ошибка проглочена
    }
```

Если битый YAML в `steps/`, `GetSteps` вернёт ошибку, но `loadProjectData` скажет «шагов нет». MCP агент получит пустой проект и может перезаписать данные.

**Фикс:** Логировать warn + возвращать частичный результат. Добавить поля `StepsError`/`DecisionsError` в `ProjectData` или логировать на уровне `ListProjects`.

**Файлы:**
- Modify: `internal/store/filestore.go:281-295`
- Test: `internal/store/store_test.go` (проверить что ошибка логируется, а не молча глотается)

**Код:**
```go
func (s *FileStore) loadProjectData(p types.Project) (*types.ProjectData, error) {
    steps, err := s.GetSteps(p.ID)
    if err != nil {
        slog.Warn("load steps for project", "project", p.ID, "error", err)
        steps = nil
    }
    decisions, err := s.GetDecisions(p.ID)
    if err != nil {
        slog.Warn("load decisions for project", "project", p.ID, "error", err)
        decisions = nil
    }
    return &types.ProjectData{Project: p, Steps: steps, Decisions: decisions}, nil
}
```

**Валидация:**
```bash
go test ./internal/store/... -v -run TestLoadProjectData
```

---

### Task 2: `uuid.Must` в CLI → правильная обработка ошибки

**Проблема:** `internal/cli/project.go:50`:
```go
id := uuid.Must(uuid.NewV7()).String()
```
`uuid.Must` паникует при ошибке. MCP хендлер обрабатывает это правильно.

**Фикс:** Заменить на `uuid.NewV7()` + return error.

**Файлы:**
- Modify: `internal/cli/project.go:50`

**Код:**
```go
uid, err := uuid.NewV7()
if err != nil {
    return fmt.Errorf("generate project id: %w", err)
}
id := uid.String()
```

---

### Task 3: `Blocker.UpdatedAt` не установлен в CLI `cmdBlockerAdd`

**Проблема:** `internal/cli/blocker.go:78-85`:
```go
blocker := types.Blocker{
    ID: id, Title: title,
    Status:    types.BlockerWaiting,
    Reason:    *reason,
    ProjectID: pd.Project.ID,
    StepID:    stepSlug,
    CreatedAt: now,  // ← UpdatedAt не установлен
}
```

MCP хендлер устанавливает `UpdatedAt: now`.

**Фикс:** Добавить `UpdatedAt: now`.

**Файлы:**
- Modify: `internal/cli/blocker.go:85` — добавить `UpdatedAt: now,`

---

### Task 4: `get_briefing` — возвращать ошибку при невалидных params

**Проблема:** `internal/mcp/tools.go:687-689`:
```go
if err := json.Unmarshal(args, &params); err != nil {
    slog.Warn("get_briefing: ignoring invalid params", "error", err)
}
```

Асимметрия с остальными хендлерами — те возвращают `fmt.Errorf("invalid args: %w", err)`. Этот глотает и генерит брифинг с пустыми параметрами.

**Фикс:** Возвращать ошибку как все.

**Файлы:**
- Modify: `internal/mcp/tools.go:687-689`

**Код:**
```go
if err := json.Unmarshal(args, &params); err != nil {
    return "", fmt.Errorf("invalid args: %w", err)
}
```

---

## Фаза 2: Безопасность данных (очередь: 5→6)

### Task 5: `DeleteProject` — корзина вместо RemoveAll

**Проблема:** `internal/store/filestore.go:198-206`:
```go
func (s *FileStore) DeleteProject(id string) error {
    ...
    return os.RemoveAll(s.projectDir(id))
}
```
MCP агент с галлюцинацией project_id может удалить проект безвозвратно. Нет `pm trash list/restore`.

**Фикс:** `os.Rename` в `.trash/<id>-<timestamp>` вместо RemoveAll. Добавить CLI команды `pm trash list`, `pm trash restore <id>`, `pm trash clean`.

**Файлы:**
- Modify: `internal/store/filestore.go` — `DeleteProject`
- Modify: `internal/store/store.go` — добавить `TrashList()`, `TrashRestore()`, `TrashClean()` в интерфейс
- Modify: `internal/store/mock.go` — добавить заглушки
- Create/Modify: `internal/cli/trash.go` — команды `pm trash list/restore/clean`
- Create/Modify: `cmd/pm/main.go` — зарегистрировать команду
- Test: `internal/store/store_test.go` — тест корзины

**Код DeleteProject:**
```go
func (s *FileStore) DeleteProject(id string) error {
    unlock, err := s.lockProject(id)
    if err != nil {
        return err
    }
    defer unlock()

    trashDir := filepath.Join(s.root, ".trash")
    if err := os.MkdirAll(trashDir, 0755); err != nil {
        return fmt.Errorf("create trash: %w", err)
    }

    src := s.projectDir(id)
    dst := filepath.Join(trashDir, fmt.Sprintf("%s-%d", id, time.Now().Unix()))
    return os.Rename(src, dst)
}
```

**Store interface changes:**
```go
type Store interface {
    // ... существующие методы
    TrashList() ([]string, error)
    TrashRestore(trashName string) error
    TrashClean() error
}
```

---

### Task 6: `NextNumber` — хранить счётчик в файле

**Проблема:** Каждый `add_project` → `ListProjects()` → `os.ReadDir` + `readProject` для каждой папки. O(n). При 100 проектах — 100 чтений.

**Фикс:** Хранить next_number в `pm/_meta/next_number`. `NextNumber()` → read + inc atomic. `SaveProject` не трогает счётчик.

**Файлы:**
- Modify: `internal/store/filestore.go` — `NextNumber()`, добавить `loadNextNumber()` / `saveNextNumber()`
- Test: проверить что NextNumber инкрементится и не регрессирует

**Код:**
```go
const metaDir = "_meta"
const nextNumFile = "next_number"

func (s *FileStore) NextNumber() (int, error) {
    dir := filepath.Join(s.root, metaDir)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return 0, fmt.Errorf("create meta dir: %w", err)
    }
    path := filepath.Join(dir, nextNumFile)
    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return 1, nil // first project
    }
    if err != nil {
        return 0, fmt.Errorf("read next number: %w", err)
    }
    n, err := strconv.Atoi(strings.TrimSpace(string(data)))
    if err != nil {
        return 1, nil // reset on corruption
    }
    return n, nil
}
```

**При создании проекта:** `SaveProject` вызывает `NextNumber()`, потом сразу инкрементит:
```go
next := number + 1
writeAtomic(path, []byte(strconv.Itoa(next)))
```

---

## Фаза 3: DRY — вынести общую бизнес-логику (очередь: 7)

### Task 7: Вынести шаговую машину в общий пакет

**Проблема:** Логика step transitions (todo → in_progress → review → done) и blocker checks продублирована в:
- `internal/cli/step.go` (cmdStepStart, cmdStepReview, cmdStepDone)
- `internal/mcp/tools.go` (handleStartStep, handleReviewStep, handleDoneStep, advanceStep)
- `internal/store/filestore.go` (SaveBlocker — логика блокировки/разблокировки)

~150 строк кода в CLI + ~120 в MCP + ~50 в store = ~320 строк для одной и той же логики.

**Фикс:** Создать `internal/domain/step.go` с функциями:
- `ValidateTransition(current, next StepStatus) error`
- `ApplyStepTransition(steps []Step, stepID string, newStatus StepStatus, validate func(Step) error) (Step, error)`
- `HasUnresolvedBlockers(blockers []Blocker) bool`

CLI и MCP хендлеры становятся тонкими обёртками.

**Файлы:**
- Create: `internal/domain/step.go`
- Modify: `internal/cli/step.go` — использовать domain
- Modify: `internal/mcp/tools.go` — использовать domain
- Test: `internal/domain/step_test.go` — все переходы

**Код примера:**
```go
package domain

import "github.com/xvantz/pm/internal/types"

var validTransitions = map[types.StepStatus][]types.StepStatus{
    types.StepTodo:        {types.StepInProgress, types.StepReview},
    types.StepInProgress:  {types.StepReview},
    types.StepReview:      {types.StepDone},
}

func ValidateTransition(current, next types.StepStatus) error {
    allowed, ok := validTransitions[current]
    if !ok {
        return fmt.Errorf("step is %s, no transitions allowed", current)
    }
    for _, a := range allowed {
        if a == next {
            return nil
        }
    }
    return fmt.Errorf("cannot transition from %s to %s", current, next)
}

func HasUnresolvedBlockers(blockers []types.Blocker) bool {
    for _, b := range blockers {
        if b.Status == types.BlockerActive || b.Status == types.BlockerWaiting {
            return true
        }
    }
    return false
}
```

---

## Фаза 4: Read-консистентность (очередь: 8)

### Task 8: `pm init` / `pm doctor` — проверка целостности

**Проблема:** Нет способа проверить хранилище на битые YAML, orphaned файлы, дублирующиеся ID.

**Фикс:** Команда `pm doctor` (`internal/cli/doctor.go`):
- Читает все `project.yaml` и проверяет парсинг
- Читает все `steps/*.yaml` и `decisions/*.yaml`
- Сверяет ProjectID в шагах/решениях с существующими проектами
- Подсчитывает orphaned файлы
- Выводит сводку: N проектов, M шагов, K блокеров, X ошибок

**Файлы:**
- Create: `internal/cli/doctor.go`
- Modify: `cmd/pm/main.go` — зарегистрировать команду
- Modify: `internal/store/filestore.go` — добавить `Doctor()` (опционально)

---

## Порядок выполнения

```
Фаза 1 — 4 quick wins, каждый ≤ 5 строк, тесты не меняются
     ↓
Фаза 2 — корзина + next_number (новые функции, тесты)
     ↓
Фаза 3 — DRY domain (рефакторинг, тесты сначала)
     ↓
Фаза 4 — pm doctor (новый код)
```

Каждая фаза самодостаточна. После любой фазы можно остановиться.

## Риски

1. **Фаза 3** — самый большой рефакторинг. CLI и MCP нужно тестировать после изменений. Рекомендуется выполнять через subagent-driven-development: сначала тесты, потом код, потом интеграция.
2. **Фаза 2 (корзина)** — меняет интерфейс `Store`. MockStore нужно обновить синхронно.
3. **Фаза 2 (next_number)** — при существующих проектах счётчик нужно инициализировать как `max(Number) + 1` при первом чтении пустого файла.

## Проверка после всех фаз

```bash
go test ./... -v -count=1
go vet ./...
go build ./cmd/pm
go build ./cmd/pm-mcp
```
