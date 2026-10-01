# Tasks: fix-env-doctor

## Phase 1: Проверка едет в демона

- [x] 1.1 `internal/types`: тип отчета (`DoctorReport`: счетчики, сироты,
  битые файлы, legacy/битые метки)
  **DoD:** `CGO_ENABLED=0 go build ./...` зелен; тип экспортирован
  **Результат:** `internal/types/doctor.go`: `DoctorReport` + `DoctorProjectLine`
  + `HasIssues()`. Build зелен.
- [x] 1.2 `internal/store`: метод проверки в интерфейсе + переезд обхода из
  `internal/cli/doctor.go` в `FileStore` (включая счетчик legacy/битых меток);
  `MockStore` - пустой отчет
  **DoD:** `CGO_ENABLED=0 go test ./internal/store/ -count=1` зелен; обход
  файлов в `internal/cli/doctor.go` отсутствует (grep `ReadDir` пуст)
  **Результат:** `Check()` в интерфейсе, `FileStore.Check` несет весь обход
  (сироты, parse/read ошибки, mismatch, legacy/broken), `MockStore` - пустой
  отчет. `grep ReadDir internal/cli/doctor.go` пуст. Старый
  `doctor_timestamp_test.go` удален, его 4 теста переехали в
  `internal/store/doctor_test.go` (5 тестов: legacy/broken/orphan+mismatch/
  missing/read-only).
- [x] 1.3 `GET /api/doctor` + проброс `client`/`apistore`; `doctor_timestamp_test.go`
  переписан под демон (FileStore напрямую + httptest)
  **DoD:** `CGO_ENABLED=0 go test ./internal/api/ ./internal/cli/ ./internal/apistore/ -count=1` зелен
  **Результат:** роут + хендлер (auth), `client.Doctor()` с ошибкой про демона,
  `apistore.Check()`. Тесты: `TestDoctorEndpoint` (200 + тело + 401 без токена),
  `TestRemoteCheck_DeadDaemonIsTheVerdict` (мертвый адрес = ошибка про демона).
- [x] 1.4 `cmdDoctor`: звать демона и печатать отчет; `doctor_live.go`: демон
  недоступен = ошибка без локального скана
  **DoD:** тест: демон молчит - `cmdDoctor` возвращает ошибку про демона, а не
  файловый отчет
  **Результат:** `doctor.go` переписан (тот же текст отчета, источник - демон),
  `defaultProjectsDir` удален как мертвый код. `TestRemoteCheck_DeadDaemonIsTheVerdict`
  доказывает: мертвый адрес = ошибка про демона.

## Phase 2: Nix и доки

- [x] 2.1 `flake.nix`: `PM_API` из `listenAddr` в sessionVariables + shellInit
  **DoD:** модуль парсится (`nix flake check` где возможно); иначе построчный
  review + DoD 3.2 как доказательство
  **Результат:** `nix-instantiate --parse flake.nix` OK. Полный eval невозможен
  из контейнера - доказательством будет rebuild на хосте (3.2).
- [ ] 2.2 Dotfiles `hermes.nix`: `env.PM_API` из `services.pm.listenAddr`
  **DoD:** хардкод `127.0.0.1:8472` в файле отсутствует (grep пуст);
  коммит отдельным PR в dotfiles
  **Статус:** правка внесена и парсится, НЕ закоммичена: дерево /dotfiles
  грязное чужими незакоммиченными правками (flake.lock + 1 строка), подметать
  их в свой коммит нельзя. Решение по коммиту - за Иваном.
- [x] 2.3 README: примеры с адресом обновлены под derived
  **DoD:** grep `127.0.0.1:8472` по README показывает только дефолт в тексте,
  не инструкцию прописывать руками
  **Результат:** run-раздел, строка `pm doctor` и пример hermes.nix обновлены.

## Phase 3: Проверка и финал

- [x] 3.1 Полный прогон: build, vet, fmt, `validate --all --strict`
  **DoD:** все зелено, 0 failed
  **Результат:** **195 тестов PASS** (было 192: +5 store, +1 api, +1 apistore,
  -4 удаленных cli), `gofmt -l` пусто, `go vet` чист, validate 8/8.
- [ ] 3.2 Rebuild на хосте + smoke: смена listenAddr не роняет клиентов
  **DoD (владелец хоста):** `nixos-rebuild switch`, свежий шелл показывает
  `PM_API` с новым адресом; `pm doctor` зелен через новый адрес, MCP отвечает
- [ ] 3.3 PR + CI + архив последним коммитом; P8/P10 пула в `[x]`
  **DoD:** Forgejo run success; `openspec list` без `fix-env-doctor`;
  baseline несет дельту

## Границы

- `PM_TOKEN` в sessionVariables не кладем. Никогда.
- `pm init` не трогаем.
- `fix-event-time` границу не трогаем: счетчик переезжает как есть.
