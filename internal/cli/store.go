package cli

import (
	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/store"
)

// openStore always talks to the daemon: remote-only, no file fallback.
// Address defaults to the serve convention (PM_API overrides), token from PM_TOKEN.
func openStore() (store.Store, error) {
	remote, _ := apistore.NewFromEnv()
	return remote, nil
}
