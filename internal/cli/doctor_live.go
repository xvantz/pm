package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/client"
)

// doctorLive answers the operational half: token, daemon, service.
// Integrity itself comes from the daemon (cmdDoctor asks it first);
// this only verifies the path to the daemon and its surroundings.
func doctorLive() error {
	fmt.Println()
	fmt.Println("PM Doctor — демон и окружение")
	fmt.Println(strings.Repeat("=", 60))

	bad := 0

	token := strings.TrimSpace(os.Getenv("PM_TOKEN"))
	noToken := token == ""
	if noToken {
		fmt.Println("  ❌ PM_TOKEN пуст: ни чтение, ни запись не пройдут.")
		fmt.Println("     Токен раздается автоматически: сервис через sops pm_env,")
		fmt.Println("     шелл и MCP из него же. Проверь: секрет pm_env на месте,")
		fmt.Println("     rebuild сделан, шелл перезапущен.")
	} else {
		fmt.Println("  ✅ PM_TOKEN задан.")
	}

	addr := strings.TrimSpace(os.Getenv("PM_API"))
	if addr == "" {
		addr = apistore.DefaultAddr
	}
	cl := client.New(addr, token)
	fmt.Printf("  Адрес демона: %s\n", addr)

	status, err := cl.Health()
	if err != nil {
		fmt.Printf("  ❌ Демон недоступен: %v\n", err)
		fmt.Println("     Подними: pm serve (или systemctl start pm-serve).")
		bad++
	} else {
		fmt.Printf("  ✅ Демон жив (healthz ok, версия %s).\n", cl.Version())
		_ = status
		if noToken {
			bad++ // counted above, probe needs a token
		} else if _, err := cl.ListProjects(); err != nil {
			fmt.Printf("  ❌ Auth не прошел: %v\n", err)
			fmt.Println("     Токен в шелле не совпал с токеном демона: сверь pm_env.")
			bad++
		} else {
			fmt.Println("  ✅ Auth прошел: проекты читаются.")
		}
	}

	checkSystemd()

	if bad > 0 {
		return fmt.Errorf("doctor found %d live problem(s)", bad)
	}
	fmt.Println()
	fmt.Println("✅ Демон и окружение в порядке.")
	return nil
}

// checkSystemd is best-effort: reports pm-serve state on NixOS hosts,
// stays silent where systemctl is absent (containers, dev machines).
func checkSystemd() {
	path, err := exec.LookPath("systemctl")
	if err != nil {
		return
	}
	out, err := exec.Command(path, "is-active", "pm-serve").Output()
	state := strings.TrimSpace(string(out))
	if err != nil || state != "active" {
		if state == "" {
			state = "unknown"
		}
		fmt.Printf("  ⚠ pm-serve systemd: %s (ожидалось active).\n", state)
		return
	}
	fmt.Println("  ✅ pm-serve systemd: active.")
}
