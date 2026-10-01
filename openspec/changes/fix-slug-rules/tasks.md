# Tasks: fix-slug-rules

## Phase 1: Спека

- [x] 1.1 Дельта `project-store` ADDED: slug contract (allowlist, кап 200 байт,
  пустота - отказ, коллизия - конфликт)
  **DoD:** `openspec validate fix-slug-rules --type change` зелен

## Phase 2: Код

- [x] 2.1 `internal/slug/slug.go`: `Of` на рунах (буквы/цифры unicode, остальное
  по правилу), кап 200 байт по границе UTF-8, трим висячего дефиса
  **DoD:** `CGO_ENABLED=0 go test ./internal/slug/ -count=1` зелен
- [x] 2.2 Граничные тесты: `! ? #` чистятся, кириллица живет, 300-символьный
  тайтл дает короткий slug, обрезка по границе руны, два длинных с общим
  префиксом коллизируют
  **DoD:** каждый пункт выше - отдельный ассерт, все зеленые
- [x] 2.3 Переписать `TestSlug_UnlimitedLength` и `TestSlug_NoCollisionOnLongTitles`:
  они утверждают отсутствие капа, что противоречит спеке
  **DoD:** ни один тест в пакете не утверждает unlimited; grep `nlimited`
  по `slug_test.go` пуст
- [x] 2.4 Существующие тесты на старые замены не сломаны (пробел, `_ / \ . : ,`,
  кавычки, скобки дают то же, что раньше)
  **DoD:** `TestSlug_SpecialChars`, `TestSlug_Cyrillic`, `TestSlug_Simple` зелены
  без правок

## Phase 3: Проверка и финал

- [x] 3.1 Коллизия end-to-end: два тайтла в один slug через API/CLI дают отказ,
  а не слияние
  **DoD:** тест или живой прогон: второй create падает с конфликтом, первый
  нетронут
- [x] 3.2 Полный прогон: build, vet, fmt, `validate --all --strict`
  **DoD:** все зелено, 0 failed
- [ ] 3.3 PR + CI + архив последним коммитом; P7 пула в `[x]`
  **DoD:** Forgejo run success; `openspec list` без `fix-slug-rules`;
  baseline несет slug contract

## Границы

- Без миграции старых ID. Файлы с `!` лежат как лежали.
- Без транслитерации и без суффиксов `-2` при коллизии.
- Валидации в api/cli/mcp не трогаем: все уже через пакет.

**Результаты:** Of на рунах (буквы/цифры unicode, пунктуация в дефис, drop-набор как раньше), кап 200 байт по границе UTF-8. Старые SpecialChars/Cyrillic зелены без правок. UnlimitedLength и часть NoCollision переписаны: grep `nlimited` пуст. Коллизия доказана `TestStepCreate_DuplicateSlugConflicts` (409, первый нетронут). 190 тестов PASS.
