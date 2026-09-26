package main

import (
	"fmt"
	"os"

	"github.com/xvantz/pm/internal/cli"
)

// Version set by -ldflags during build; fallback for dev.
var Version = "dev"

func main() {
	cli.Version = Version
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
