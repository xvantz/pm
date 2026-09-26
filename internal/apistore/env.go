package apistore

import (
	"os"
	"strings"

	"github.com/xvantz/pm/internal/client"
)

// ClientFromEnv returns a client when PM_API is set.
func ClientFromEnv() (*client.Client, bool) {
	addr := strings.TrimSpace(os.Getenv("PM_API"))
	if addr == "" {
		return nil, false
	}
	return client.New(addr, os.Getenv("PM_TOKEN")), true
}

// NewFromEnvVars builds a Store from an explicit API address and token.
// It reports false when addr is empty, so callers fall back to the file store.
func NewFromEnvVars(addr, token string) (*Store, bool) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, false
	}
	return New(client.New(addr, token)), true
}

func envAPI() string {
	if v := os.Getenv("PM_API"); v != "" {
		return v
	}
	return ""
}

func envToken() string {
	return os.Getenv("PM_TOKEN")
}
