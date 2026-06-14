// pm-mcp is an MCP server that exposes PM (Project Memory) tools over stdio.
//
// It enables LLM agents to read, create, and manage project memory data
// through the Model Context Protocol.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/xvantz/pm/internal/mcp"
	"github.com/xvantz/pm/internal/store"
)

func main() {
	root := projectsDir()

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "projects dir not found: %s\n  Run `pm init` first.\n", root)
		os.Exit(1)
	}

	st := store.NewFileStore(root)

	server := mcp.NewServer("pm-mcp", "0.1.0")
	mcp.RegisterPMTools(server, st)

	slog.Info("PM MCP server started", "dir", root)
	if err := server.Run(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func projectsDir() string {
	if dir := os.Getenv("PM_DIR"); dir != "" {
		return dir
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "./pm/projects"
	}
	return filepath.Join(cwd, "pm", "projects")
}
