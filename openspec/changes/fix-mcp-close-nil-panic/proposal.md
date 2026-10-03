# Proposal: fix-mcp-close-nil-panic

## Why

Прод-паника с живым репродом (2026-10-03): `close_project` с `confirm: true`
через демона убивает весь pm-mcp процесс (SIGSEGV, `tools.go:1098`,
`plan.StepsToClose()` на nil). Закрытие при этом успевает выполниться -
агент видит смерть транспорта, проект уже completed. Обнаружено на реальном
закрытии #5, воспроизведено локально на сборке из main.

Цепочка расхождения мок/демон:

1. `MockStore.CloseProject` на confirm возвращает `(plan, nil)` - план жив.
2. Демон на confirm возвращает `{Project, Plan: nil}` (`server.go:318`,
   контракт "never both" в комментарии к `closeProjectResp`).
3. Client/apistore честно пробрасывают nil дальше.
4. `handleCloseProject` безусловно зовет `plan.StepsToClose()` - бум.

Все unit-тесты ходят через мок, поэтому зелены, а прод падает. Тестовая
традиция репо (httptest-демон вместо мока там, где поведение зависит от
демона) на этот путь не распространялась.

## What Changes

- **Сервер на confirm тоже отдает план**: строит через `ClosePlan` до
  закрытия, под тем же `s.mu`, возвращает `{Confirmed, Project, Plan}`.
  Контракт "never both" меняется на "на confirm оба". Client и apistore
  уже пробрасывают план как есть (`out.Plan != nil` первым) - их трогать не
  надо.
- **Nil-guard в хендлере второй линией**: confirm-путь не дереференсит plan
  без проверки; при nil считает moved как 0... нет - врет. Правильно: при
  nil план сообщение без счетчика не врет, но теряет информацию. После
  серверного фикса план на confirm всегда есть; guard - только чтобы паника
  никогда не повторилась ни при каком сторе: `moved` из плана при наличии,
  иначе текст "closed" без числа.
- **Регресс-тест через httptest-демона** (паттерн `TestDoctorEndpoint`):
  confirm через apistore поверх тестового сервера - падает на текущем коде
  паникой, зелен после фикса. Мок-тест остается как есть.

## Non-goals

- Не меняем двухвызовный consent-флоу и тексты.
- Не чиним клиентские таймауты/переподключения Hermes - падение было нашей
  паникой, транспорт в порядке.
- Не переписываем остальные мок-тесты на httptest оптом - только этот путь.

## Impact

- Affected specs: `serve-api` (MODIFIED: confirm несет и проект, и план),
  `mcp-server` (ADDED: хендлер переживает nil plan).
- Affected code: `internal/api/server.go` (confirm-ответ + комментарий),
  `internal/mcp/tools.go` (guard), `internal/mcp/close_consent_test.go`
  или новый тест (регресс через httptest).
