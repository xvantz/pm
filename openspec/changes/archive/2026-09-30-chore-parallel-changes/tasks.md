# Tasks: chore-parallel-changes (DoD — команда или наблюдение)

## Phase 1: Правило

- [x] 1.1 `AGENTS.md` правило 3: «по одному» → «разные файлы или worktree»,
  с проверяемым признаком (пересечение `Affected code`), границы не меняются
  **DoD:** в `AGENTS.md` нет формулировки «one change fully closed before the next
  opens»; правило 3 содержит термин «worktree» и критерий пересечения файлов
  **Результат:** правило 3 переписано: гейт — пересечение файлов, а не порядок.
  Формулировка «one change fully closed before the next opens» удалена. Нумерация
  сплошная 1-7, кросс-проверка фактов ниже.
- [x] 1.2 Добавить правило verify перед merge (baseline-спека против дельты соседа)
  **DoD:** в `AGENTS.md` есть отдельный пункт про verify соседей перед merge,
  не дублирующий close definition
  **Результат:** новое правило 4 (verify соседей грепом по baseline перед merge),
  отделено от close definition (теперь правило 7).
- [x] 1.3 Перепроверить правила 1, 2, 4, 5, 6 — не должны пострадать при правке
  **DoD:** все шесть правил на месте, нумерация сплошная, pre-commit/pre-push и CI
  в AGENTS.md совпадают с фактическими путями (`scripts/hooks`, `.forgejo/workflows/ci.yml`)
  **Результат:** правила 1, 2 сохранены дословно; 4, 5, 6 сдвинуты на +1 из-за
  вставки verify. Пути сверены с фактом: `scripts/hooks/{pre-commit,pre-push}`
  существуют, `.forgejo/workflows/ci.yml` использует checkout@v7 + setup-go@v7,
  pre-push имеет CGO_ENABLED=0 и пропуск delete-only пушей. Старое правило 6
  (close definition) сохранено как правило 7, добавлено «архив последним
  коммитом того же PR».
  **Побочно найдено:** ни у одного из 6 открытых ченжей (пул, paging, trash,
  force, env-doctor, slug) нет ни одного `DoD:` в tasks.md — правило 2 нарушено
  ими, а не этим ченджем. Не чинил молча: это отдельная работа (см. proposal).

## Phase 2: Проверка

- [x] 2.1 Отметить границы в трёх MCP-ченджах (`Affected code` пересекается)
  **DoD:** в proposal `feat-mcp-paging-briefing` и `feat-mcp-trash-restore` есть
  строка о том, что они делят `internal/mcp/tools.go`, и порядок (кто мерджится первым)
  **Результат:** во всех трёх (`paging`, `trash-restore`, `step-lifecycle-force`)
  добавлен блок «Пересечение файлов» с указанием делящего файла и порядка мерджа:
  paging → trash-restore → step-lifecycle-force. Отмечено, что force правит только
  схему `done_step`, поэтому конфликта требований нет, только файла.

## Границы

- Кода нет. Только процесс-правило и текст в чужих proposal.
- Не добавлять автоматический gate на пересечение файлов: это false-positive-генератор.
- Не менять close definition: он про «когда ченж закончен», а не про количество.
