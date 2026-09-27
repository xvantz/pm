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
// File integrity is checked by cmdDoctor before this runs.
func doctorLive() error {
	fmt.Println()
	fmt.Println("PM Doctor — демон и окружение")
	fmt.Println(strings.Repeat("=", 60))

	bad := 0

	token := strings.TrimSpace(os.Getenv("PM_TOKEN"))
	if token == "" {
		fmt.Println("  ⚠ PM_TOKEN пуст: с loopback пустит без токена, из сети будет 401.")
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
		if _, err := cl.ListProjects(); err != nil {
			fmt.Printf("  ❌ Проекты не читаются: %v\n", err)
			fmt.Println("     Loopback без токена закрыт, а PM_TOKEN не подошел: сверь с токеном демона.")
			bad++
		} else if token == "" {
			fmt.Println("  ✅ Чтение без токена: loopback доверенный.")
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
