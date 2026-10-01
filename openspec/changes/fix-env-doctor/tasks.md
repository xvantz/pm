# Tasks: fix-env-doctor

## Phase 1: Проверка едет в демона

- [ ] 1.1 `internal/types`: тип отчета (`DoctorReport`: счетчики, сироты,
  битые файлы, legacy/битые метки)
  **DoD:** `CGO_ENABLED=0 go build ./...` зелен; тип экспортирован
- [ ] 1.2 `internal/store`: метод проверки в интерфейсе + переезд обхода из
  `internal/cli/doctor.go` в `FileStore` (включая счетчик legacy/битых меток);
  `MockStore` - пустой отчет
  **DoD:** `CGO_ENABLED=0 go test ./internal/store/ -count=1` зелен; обход
  файлов в `internal/cli/doctor.go` отсутствует (grep `ReadDir` пуст)
- [ ] 1.3 `GET /api/doctor` + проброс `client`/`apistore`; `doctor_timestamp_test.go`
  переписан под демон (FileStore напрямую + httptest)
  **DoD:** `CGO_ENABLED=0 go test ./internal/api/ ./internal/cli/ ./internal/apistore/ -count=1` зелен
- [ ] 1.4 `cmdDoctor`: звать демона и печатать отчет; `doctor_live.go`: демон
  недоступен = ошибка без локального скана
  **DoD:** тест: демон молчит - `cmdDoctor` возвращает ошибку про демона, а не
  файловый отчет

## Phase 2: Nix и доки

- [ ] 2.1 `flake.nix`: `PM_API` из `listenAddr` в sessionVariables + shellInit
  **DoD:** модуль парсится (`nix flake check` где возможно); иначе построчный
  review + DoD 3.2 как доказательство
- [ ] 2.2 Dotfiles `hermes.nix`: `env.PM_API` из `services.pm.listenAddr`
  **DoD:** хардкод `127.0.0.1:8472` в файле отсутствует (grep пуст);
  коммит отдельным PR в dotfiles
- [ ] 2.3 README: примеры с адресом обновлены под derived
  **DoD:** grep `127.0.0.1:8472` по README показывает только дефолт в тексте,
  не инструкцию прописывать руками

## Phase 3: Проверка и финал

- [ ] 3.1 Полный прогон: build, vet, fmt, `validate --all --strict`
  **DoD:** все зелено, 0 failed
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
