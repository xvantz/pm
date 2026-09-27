# Why

Деплой и source разъехались по транспорту. Задеплоенный бинарь (rev `f22339ff`) говорит NDJSON (проверено: в бинаре нет строки `Content-Length`, handshake с Hermes проходит). Текущий source на main (`server.go:226` `writeMessage`, `messageReader.readMessage`) пишет и требует Content-Length framing. Значит фикс схемы из предыдущего ченжа не спасет, если его задеплоить с текущего main: транспорт, который Hermes Python-клиент не парсит, даст новую аварию вместо старой. Один фикс без второго - стрельба себе в ногу.

# What Changes

- Решение (одно из двух, spike первым коммитом): (A) откатить `server.go` на NDJSON построчный обмен, зафиксировав interop-тест байтами реального Hermes handshake; или (B) перейти на официальный MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`) и выкинуть hand-rolled `messageReader`/`writeMessage`/`ID *int`.
- Рекомендация: (B). Свои `ID *int` ломаются на string ID, нет `ping`, нет `notifications/cancelled` - каждый апдейт Hermes будет лотереей.
- Тест: золотая запись `initialize -> notifications/initialized -> tools/list -> tools/call` байт-в-байт против формата, который понимает Hermes.

# Impact

- Affected specs: `mcp-server` (MODIFIED: требование транспорта).
- Affected code: `internal/mcp/server.go`, `internal/mcp/mcp_test.go`, `go.mod` (при варианте B).
- Блокирует любой редеплой с main. Идет строго после `fix-mcp-list-projects-schema`, деплоятся вместе.
