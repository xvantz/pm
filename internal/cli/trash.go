package cli

import (
	"fmt"
	"strings"
)

func cmdTrash(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm trash <list|restore|clean>")
	}

	switch args[0] {
	case "list":
		return cmdTrashList(args[1:])
	case "restore":
		return cmdTrashRestore(args[1:])
	case "clean":
		return cmdTrashClean(args[1:])
	default:
		return fmt.Errorf("unknown trash subcommand: %s", args[0])
	}
}

func cmdTrashList(args []string) error {
	st, err := openStore()
	if err != nil {
		return err
	}
	items, err := st.TrashList()
	if err != nil {
		return fmt.Errorf("list trash: %w", err)
	}
	if len(items) == 0 {
		fmt.Println("Trash is empty.")
		return nil
	}
	fmt.Println("Trash items:")
	for _, item := range items {
		fmt.Printf("  %s\n", item)
	}
	return nil
}

func cmdTrashRestore(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm trash restore <trash-name>")
	}
	st, err := openStore()
	if err != nil {
		return err
	}
	trashName := strings.Join(args, " ")
	if err := st.TrashRestore(trashName); err != nil {
		return fmt.Errorf("restore %q: %w", trashName, err)
	}
	fmt.Printf("Restored %q.\n", trashName)
	return nil
}

func cmdTrashClean(args []string) error {
	st, err := openStore()
	if err != nil {
		return err
	}
	if err := st.TrashClean(); err != nil {
		return fmt.Errorf("clean trash: %w", err)
	}
	fmt.Println("Trash emptied.")
	return nil
}
