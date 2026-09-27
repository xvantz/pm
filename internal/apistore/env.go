package apistore

import (
	"os"
	"strings"

	"github.com/xvantz/pm/internal/client"
)

// DefaultServeAddr is the host:port `pm serve` listens on by default.
// DefaultAddr is its http URL. Single convention, declared once.
const DefaultServeAddr = "127.0.0.1:8472"

// DefaultAddr is the daemon address used when PM_API is unset.
const DefaultAddr = "http://" + DefaultServeAddr

// ClientFromEnv returns a client for PM_API, falling back to DefaultAddr.
// Token comes from PM_TOKEN (empty means the daemon rejects everything
// except /healthz, and the error surfaces on first call).
func ClientFromEnv() (*client.Client, bool) {
	addr := strings.TrimSpace(os.Getenv("PM_API"))
	if addr == "" {
		addr = DefaultAddr
	}
	return client.New(addr, os.Getenv("PM_TOKEN")), true
}

// NewFromEnvVars builds a Store from an explicit API address and token.
// Empty addr falls back to DefaultAddr; it never reports false so callers
// don't need a file-store fallback at all.
func NewFromEnvVars(addr, token string) (*Store, bool) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = DefaultAddr
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
