// pm-mcp is an MCP server that exposes PM (Project Memory) tools over stdio.
//
// It enables LLM agents to read, create, and manage project memory data
// through the Model Context Protocol.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/mcp"
	"github.com/xvantz/pm/internal/store"
)

// Version set by -ldflags during build; fallback for dev.
var Version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("pm-mcp version %s\n", Version)
		return
	}

	// Remote-only: no files here, everything goes to the daemon.
	// Address defaults to the serve convention (PM_API overrides), token from PM_TOKEN.
	var st store.Store
	remote, _ := apistore.NewFromEnv()
	st = remote
	if api := os.Getenv("PM_API"); api != "" {
		slog.Info("PM MCP server started in remote mode", "api", api)
	} else {
		slog.Info("PM MCP server started in remote mode", "api", apistore.DefaultAddr)
	}

	server := mcp.NewServer("pm-mcp", Version)
	mcp.RegisterPMTools(server, st)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := server.Run(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
