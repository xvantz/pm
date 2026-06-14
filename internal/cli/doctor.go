package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/xvantz/pm/internal/types"
)

func cmdDoctor(args []string) error {
	root := defaultProjectsDir()

	// Check if store exists
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		fmt.Println("PM Doctor — проверка целостности хранилища")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("❌ Хранилище не найдено: %s\n", root)
		fmt.Println("   Запустите `pm init`.")
		return nil
	}

	fmt.Println("PM Doctor — проверка целостности хранилища")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Путь: %s\n\n", root)

	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read store root: %w", err)
	}

	known := map[string]bool{".trash": true, "_meta": true}
	var totalProjects, totalSteps, totalDecisions, totalBlockers int
	var errCount int
	var orphans []string

	for _, e := range entries {
		if !e.IsDir() || known[e.Name()] {
			continue
		}

		projDir := filepath.Join(root, e.Name())
		projectFile := filepath.Join(projDir, "project.yaml")

		if _, err := os.Stat(projectFile); os.IsNotExist(err) {
			orphans = append(orphans, e.Name())
			continue
		}

		data, err := os.ReadFile(projectFile)
		if err != nil {
			fmt.Printf("  ❌ %s: read error: %v\n", e.Name(), err)
			errCount++
			continue
		}
		var p types.Project
		if err := yaml.Unmarshal(data, &p); err != nil {
			fmt.Printf("  ❌ %s: YAML parse error: %v\n", e.Name(), err)
			errCount++
			continue
		}

		fmt.Printf("  ✅ #%d %s (%s)\n", p.Number, p.Title, p.ID)
		totalProjects++

		// Check steps
		stepsDir := filepath.Join(projDir, "steps")
		stepCount := 0
		stepBlockers := 0
		if stepEntries, err := os.ReadDir(stepsDir); err == nil {
			for _, se := range stepEntries {
				if se.IsDir() || filepath.Ext(se.Name()) != ".yaml" {
					continue
				}
				stepCount++
				stepPath := filepath.Join(stepsDir, se.Name())
				stepData, err := os.ReadFile(stepPath)
				if err != nil {
					fmt.Printf("     ⚠ step %s: read error: %v\n", se.Name(), err)
					errCount++
					continue
				}
				var step types.Step
				if err := yaml.Unmarshal(stepData, &step); err != nil {
					fmt.Printf("     ⚠ step %s: YAML parse error: %v\n", se.Name(), err)
					errCount++
					continue
				}
				if step.ProjectID != p.ID {
					fmt.Printf("     ⚠ step %s: project_id mismatch (%s != %s)\n", se.Name(), step.ProjectID, p.ID)
					errCount++
				}
				stepBlockers += len(step.Blockers)
			}
		}
		totalSteps += stepCount
		totalBlockers += stepBlockers

		// Check decisions
		decDir := filepath.Join(projDir, "decisions")
		decCount := 0
		if decEntries, err := os.ReadDir(decDir); err == nil {
			for _, de := range decEntries {
				if de.IsDir() || filepath.Ext(de.Name()) != ".yaml" {
					continue
				}
				decCount++
				decPath := filepath.Join(decDir, de.Name())
				decData, err := os.ReadFile(decPath)
				if err != nil {
					fmt.Printf("     ⚠ decision %s: read error: %v\n", de.Name(), err)
					errCount++
					continue
				}
				var dec types.Decision
				if err := yaml.Unmarshal(decData, &dec); err != nil {
					fmt.Printf("     ⚠ decision %s: YAML parse error: %v\n", de.Name(), err)
					errCount++
					continue
				}
				if dec.ProjectID != p.ID {
					fmt.Printf("     ⚠ decision %s: project_id mismatch (%s != %s)\n", de.Name(), dec.ProjectID, p.ID)
					errCount++
				}
			}
		}
		totalDecisions += decCount

		fmt.Printf("     Шагов: %d, Блокеров: %d, Решений: %d\n", stepCount, stepBlockers, decCount)
	}

	// Summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Итого:\n")
	fmt.Printf("  Проектов:  %d\n", totalProjects)
	fmt.Printf("  Шагов:     %d\n", totalSteps)
	fmt.Printf("  Блокеров:  %d\n", totalBlockers)
	fmt.Printf("  Решений:   %d\n", totalDecisions)

	if len(orphans) > 0 {
		fmt.Printf("\n⚠ Сиротских директорий (без project.yaml): %d\n", len(orphans))
		for _, o := range orphans {
			fmt.Printf("  - %s\n", o)
		}
	}

	if errCount > 0 {
		fmt.Printf("\n❌ Найдено ошибок: %d\n", errCount)
		return fmt.Errorf("doctor found %d error(s)", errCount)
	}

	fmt.Println()
	fmt.Println("✅ Хранилище в порядке.")
	return nil
}
