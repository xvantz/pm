# Tasks: fix-env-doctor

## Phase 1: Демон и клиенты

- [ ] 1.1 `GET /healthz` отдает `data_dir`; `client.Health` пробрасывает поле
  **DoD:** тест: healthz JSON содержит `data_dir` с корнем стора; старые
  клиенты (декод в `map[string]string`) не ломаются
- [ ] 1.2 Doctor: печатает file-сторону и daemon-сторону, расхождение корней -
  варнинг с обеими сторонами
  **DoD:** тест с фикстурой: `PM_DIR=/a`, демон на `/b` - вывод содержит обе
  стороны и слово mismatch; совпадение - варнинга нет

## Phase 2: Nix и доки

- [ ] 2.1 `flake.nix`: `PM_API` из `listenAddr` в sessionVariables + shellInit
  **DoD:** `nix` парсит модуль без ошибок (`nix flake check` где возможно);
  иначе построчный review диффа + rebuild на хосте как DoD 2.4
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
  `PM_API` с новым адресом; `pm briefing` и MCP отвечают через новый адрес
- [ ] 3.3 PR + CI + архив последним коммитом; P8/P10 пула в `[x]`
  **DoD:** Forgejo run success; `openspec list` без `fix-env-doctor`;
  baseline несет дельту

## Границы

- `PM_TOKEN` в sessionVariables не кладем. Никогда.
- Демон не валидирует адрес при старте; smoke - дело хоста.
- `fix-event-time` границу не трогаем.
