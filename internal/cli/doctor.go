package cli

import (
	"fmt"
	"strings"
)

// cmdDoctor prints the daemon's integrity verdict.
//
// It asks the daemon instead of walking files itself: the daemon is the
// single reader, so there is nothing to compare and nothing to diverge.
// A dead daemon is the verdict (fix the daemon) - there is deliberately no
// local fallback scan, which would reintroduce the second reader.
func cmdDoctor(args []string) error {
	st, err := openStore()
	if err != nil {
		return err
	}
	rep, err := st.Check()
	if err != nil {
		if strings.Contains(err.Error(), "store not found") {
			fmt.Println("PM Doctor — проверка целостности хранилища")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Printf("❌ %v\n", err)
			fmt.Println("   Запустите `pm init`.")
			return nil
		}
		return err
	}

	fmt.Println("PM Doctor — проверка целостности хранилища")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Путь (взгляд демона): %s\n\n", rep.Root)

	for _, p := range rep.Projects {
		fmt.Printf("  ✅ #%d %s (%s)\n", p.Number, p.Title, p.ID)
		fmt.Printf("     Шагов: %d, Блокеров: %d, Решений: %d\n", p.Steps, p.Blockers, p.Decisions)
	}
	for _, issue := range rep.Issues {
		fmt.Printf("  ❌ %s\n", issue)
	}

	// Legacy timestamps: readable, but date-only, so they carry no intraday
	// information. They are rewritten in canonical form the next time the
	// record is written, so this is a progress counter, not a repair list.
	fmt.Println()
	fmt.Printf("Метки времени: %d устаревших (только дата), %d битых\n",
		rep.LegacyTimestamps, rep.BrokenTimestamps)
	if rep.LegacyTimestamps > 0 {
		fmt.Println("  Устаревшие метки перепишутся в RFC3339 при следующем изменении записи.")
	}
	if rep.BrokenTimestamps > 0 {
		fmt.Println("  ⚠ Битые метки не читаются: брифинг исключает их из подсчётов и пишет warning.")
	}

	// Summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Итого:\n")
	fmt.Printf("  Проектов:  %d\n", len(rep.Projects))
	fmt.Printf("  Шагов:     %d\n", rep.TotalSteps)
	fmt.Printf("  Блокеров:  %d\n", rep.TotalBlockers)
	fmt.Printf("  Решений:   %d\n", rep.TotalDecisions)

	if len(rep.Orphans) > 0 {
		fmt.Printf("\n⚠ Сиротских директорий (без project.yaml): %d\n", len(rep.Orphans))
		for _, o := range rep.Orphans {
			fmt.Printf("  - %s\n", o)
		}
	}

	if rep.HasIssues() {
		fmt.Printf("\n❌ Найдено ошибок: %d\n", len(rep.Issues)+len(rep.Orphans))
		return fmt.Errorf("doctor found %d issue(s)", len(rep.Issues)+len(rep.Orphans))
	}

	fmt.Println()
	fmt.Println("✅ Хранилище в порядке.")

	return doctorLive()
}
