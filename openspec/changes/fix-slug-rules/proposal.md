# Proposal: fix-slug-rules

## Why

Правила slug нигде не специфицированы (P7 пула): `! ? #` не чистятся, длины нет, `"..."` дает скрытые файлы. ID файловые, сюрпризы там опасны.

## What Changes

- Спека правил: разрешенный набор символов, max длина, запрет ведущих точек, нормализация коллизий (`Hello!` vs `Hello`).
- Код `internal/slug` под спеку + тесты на граничные случаи.
- Существующие ID не мигрируются (обратная совместимость файлов).

## Impact

- Affected specs: `project-store` (ADDED: slug contract) или новая capability при триаже.
- Affected code: `internal/slug`, валидации в api/cli/mcp (единая точка).
