package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

func Run(args []string) error {
	if len(args) < 1 {
		PrintUsage()
		return nil
	}

	cmd := args[0]
	switch cmd {
	case "briefing":
		return cmdBriefing(args[1:])
	case "init":
		return cmdInit(args[1:])
	case "project":
		return cmdProject(args[1:])
	case "step":
		return cmdStep(args[1:])
	case "blocker":
		return cmdBlocker(args[1:])
	case "decision":
		return cmdDecision(args[1:])
	case "add":
		return cmdAdd(args[1:])
	case "del":
		return cmdDel(args[1:])
	case "help", "--help", "-h":
		PrintUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s\nRun `pm help` for usage.", cmd)
	}
}

func PrintUsage() {
	fmt.Println(`Project Memory (PM) — долговременная память проектов.

Usage:
  pm init                        создать pm-репозиторий
  pm add project <title>         создать проект
  pm add step <proj> <title>     добавить шаг
  pm add blocker [--reason R] <proj> <step> <title>  добавить блокер
  pm add decision [--reason R] <proj> <title>         записать решение
  pm del project <id>            удалить проект
  pm del step <proj> <step>      удалить шаг
  pm del blocker <proj> <step> <blk>  удалить блокер
  pm del decision <proj> <dec>   удалить решение
  pm project list                список проектов
  pm project show <id>           детали проекта
  pm project goal <id> <text>    установить цель проекта
  pm project tag <id> <tag>...   добавить теги проекту
  pm project status <id> <st>    изменить статус (idea|active|paused|completed)
  pm step start <id> <step-id>   начать работу над шагом
  pm step review <id> <step-id>  отправить шаг на ревью (агент)
  pm step done <id> <step-id>    завершить шаг (только с ревью)
  pm step list <id>              список шагов проекта
  pm blocker resolve <proj> <step> <blk>  снять блокер
  pm blocker list <id>           список блокеров проекта
  pm decision list <id>          список решений
  pm briefing [flags]            показать брифинг
  pm help                        эта справка

Briefing flags:
  --mock          использовать демо-данные
  --date YYYY-MM-DD  брифинг на дату
  --dir path      путь к projects/ (по умолчанию ./pm/projects)
  --json          вывод в JSON
  --project N     брифинг по одному проекту (номер или UUID)

Examples:
  pm init
  pm project create "AdGuard Home"
  pm project goal 1 "Развернуть DNS-сервер"
  pm project tag 1 infrastructure homelab
  pm project list
  pm briefing`)
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
