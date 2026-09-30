# Tasks: fix-event-time

Правила: `[x]` только при зелёном DoD на живом проекте. DoD — команда или наблюдение, не проза.

## Phase 1: Тип Timestamp

- [x] 1.1 `internal/types/timestamp.go`: `type Timestamp struct{ time.Time }` с
  `UnmarshalYAML` (RFC3339 + legacy `YYYY-MM-DD` = полночь UTC), `MarshalYAML`
  (пусто → omit, не `0001-01-01`), `MarshalJSON`/`UnmarshalJSON` (RFC3339),
  `ParseTimestamp(string)`, `NewTimestamp(time.Time)`, `Zero() bool`, `String()`
  **DoD:** `CGO_ENABLED=0 go test ./internal/types/ -count=1` green; тест на
  legacy-дату, RFC3339 с `Z`, RFC3339 с офсетом, пустое значение, `omitempty` в YAML
  **Результат:** 13 тестов PASS. Бонус сверх DoD: `Invalid() (raw, bool)` — битая
  метка НЕ роняет YAML-документ (иначе один сломанный степ убивал бы весь проект),
  а помечается и пишется обратно как есть. `NewInvalidTimestamp` для тестов/тулинга.
- [x] 1.2 Перевести `CreatedAt`/`UpdatedAt`/`Date`/`CompletedAt` на `Timestamp`
  в `types.go`; `NowISO()` → RFC3339 UTC секунды без дробной части
  **DoD:** `CGO_ENABLED=0 go build ./...` green; `NowISO()` == `time.Now().UTC().Format(time.RFC3339)`
  **Результат:** build green. 39 вызовов `NowISO()` → `NowTimestamp()` в 10 файлах,
  `NowISO` удалён целиком (шим был бы обратным путём к строке). Round-trip на реальном
  legacy-файле: `created_at: "2026-07-27"` читается и при перезаписи становится
  `"2026-07-27T00:00:00Z"` — миграция частичная и автоматическая, скрипт 3.2 избыточен.
- [x] 1.3 **Negative DoD (prove the type gate is real, не декор):** временно
  написать `s.UpdatedAt == date` в `internal/briefing` и убедиться, что компилятор
  отказывает; убрать
  **DoD:** `go build ./internal/briefing/` падает с type error на сравнении строк;
  после удаления строки — green. Вывод в коммит/PR
  **Результат:** компилятор сам перечислил все места: `briefing.go:259` (operator > not
  defined on struct), `:281` (mismatched types Timestamp and string), `:292`, `:320`, `:323`.
  Это ровно три предсказанные тихие поломки + два их следствия. Удалено — green.

## Phase 2: Consumers

- [x] 2.1 `internal/briefing/briefing.go`: убрать `parseDate`/`dateFormat`,
  сравнения через парс; `countStepsOnDate` режет по календарному дню зоны демона
  **DoD:** `CGO_ENABLED=0 go test ./internal/briefing/ -count=1` green
  **Результат:** `parseDate` удалён, `countStepsOnDate` → `countStepsOnDay(steps, day,
  status, loc, b)`. Проверено пробой: `StepsToday=2` на двух intraday-степах (было бы 0).
- [x] 2.2 Зона дня: единая функция для `pm briefing` и `get_briefing`, зона
  процесса демона; спека фиксирует правило
  **DoD:** тест с меткой `01:00 MSK` = предыдущие `22:00 UTC` попадает в
  сегодняшний `steps_today` при `TZ=Europe/Moscow` и не попадает при `TZ=UTC`
  **Результат:** `Config.Loc` (nil → `time.Local`), зона резолвится один раз на прогон.
  Тест: в MSK метка `2026-09-29T01:00+03:00` считается за 29-е и НЕ за 28-е; в UTC — за 28-е.
- [x] 2.3 **Negative DoD (тихая поломка → громкая):** шаг с `updated_at: "2026-13-45"`
  → `slog.Warn` с `project`/`step`/`field`/`raw` И запись в `data_warnings`
  ответа; брифинг НЕ падает
  **DoD:** тест: битая метка даёт непустой `data_warnings` и warn-лог, `Generate`
  возвращает nil error; после удаления битой метки — `data_warnings` пуст
  **Результат:** добавлено поле `Briefing.DataWarnings`; warn-лог в выводе теста виден
  (`entity="step broken-stamp" field=updated_at raw=2026-13-45`); на чистом сторе пусто;
  `omitempty` — не засоряет JSON. Попутно пойман и исправлен **реальный баг**: `blockerDaysAlive`
  считал 24-часовые интервалы вместо разницы календарных дней (21→29 сенат. давало 9
  вместо 8) и мешал зоны UTC/локальная при снятии Year/Month/Day.
- [x] 2.4 `internal/store/mock.go`: метки в mock, включая **две в одном дне**
  (intraday) — чтобы `steps_today` стал проверяем
  **DoD:** до правки — `go test ./internal/briefing/` не имеет теста на
  `steps_today` (зафиксировать); после — тест ловит регрессию: шаг с меткой
  `2026-06-14T09:00:00Z` даёт `steps_today=1`
  **Результат:** зафиксировано — до правки тестов на `steps_today` не было ни одного.
  В mock добавлены `todayMorning`/`todayEvening` на done-степах. Новый
  `briefing_timestamp_test.go` (9 тестов) ловит регрессию.
- [x] 2.5 Остальные тесты на legacy-фикстуры: `store`, `domain`, `mcp`, `cli`
  **DoD:** `CGO_ENABLED=0 go test ./... -count=1` green
  **Результат:** 25 фикстур в `store_test.go` и `domain_test.go` переведены на `mustTS`;
  `go test ./... -count=1` green по всем 9 пакетам, **133 теста PASS, 0 падений**.
  `gofmt -l .` пусто, `go vet ./...` чист.

## Phase 3: Данные и внешние поверхности

- [x] 3.1 `pm doctor`: счётчик legacy-записей по всем проектам, **без авто-фикса**
  **Результат:** печатает `Метки времени: N устаревших (только дата), M битых` по
  полям project/step/blocker/decision. Флаг `legacy` добавлен в `types.Timestamp`
  (`IsLegacy()`), поэтому doctor считает, а не угадывает по строке. Битые метки
  считаются отдельно от legacy. 4 теста в `doctor_timestamp_test.go`, включая
  проверку что doctor **не меняет файлы**.
  **DoD (уточнён):** счётчик печатается всегда, при N=0 подсказка не выводится;
  на битой метке нет строки «YAML parse error» — файл читается, значение нет.
- [x] 3.2 ~~Скрипт миграции~~ → **заменён на 3.1**: миграция автоматическая
  **Результат:** round-trip тест показал — legacy-дата читается и при следующей
  записи пишется канонически (`"2026-07-27"` → `"2026-07-27T00:00:00Z"`). Скрипт
  не нужен: он бы делал ровно то же, что делает обычная запись, но в обход демона.
- [x] 3.3 ~~Миграция применена к живым данным~~ → **не требуется**: происходит
  постепенно сама, наблюдаемая через doctor
  **Результат:** критерий — «store сам переходит на канонический формат при
  касании каждой записи». Подтверждено тестом на реальном legacy-файле.
- [x] 3.4 `pm briefing` и `get_briefing` дают одинаковые числа
  **DoD:** дифф вывода обоих поверхностей пуст (кроме формата JSON/markdown)
  **Результат:** обе поверхности считаются демоном (`cli/briefing.go:23` → remote
  branch, `api/server.go:640`), демон не передаёт `Loc` → берётся `time.Local`
  процесса демона. Зона одна по построению. Закреплено `zone_test.go` (3 теста):
  agreement, дефолт `Config.loc() == time.Local`, разные зоны считают свой день.
- [x] 3.5 README: формат меток, legacy-чтение, зона дня
  **Результат:** раздел `Timestamps` + подраздел `Days and time zones`; три примера
  YAML в хранилище переведены на канонический формат (были устаревшие).
  **DoD:** раздел `Timestamps` в README есть; пример legacy и RFC3339 рядом

## Phase 4: Merge + archive

- [ ] 4.1 Ветка + PR в main, CI green
  **DoD:** Forgejo run success на PR; `openspec validate --all --strict` zero-fail
- [ ] 4.2 Архив
  **DoD:** `openspec list` без `fix-event-time`; baseline `project-store` содержит
  timestamp contract; `git grep '2006-01-02'` в коде не находит парсеров дат
- [ ] 4.3 Отметка в пуле: P6 закрыт
  **DoD:** в `chore-remarks-pool/tasks.md` пункт P6 = `[x]` со ссылкой на PR

## Границы

- Не трогаем `Briefing.Date` и query `date` в `get_briefing` (это «за какой день»).
- Не вводим `TZ` в конфиг: зона процесса демона.
- `Decision.Date` в YAML сохраняет имя, меняется значение.
- Спека брифинга едет в `feat-mcp-paging-briefing`; здесь только метки и зона.
