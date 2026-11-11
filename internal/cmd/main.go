package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

// Main is the process entry point shared by every main package: the
// canonical ./cmd/pfix, and the module-root package kept so that
// `go install github.com/a68366/pfix-cli@latest` keeps working. It cancels
// the command context on Ctrl-C, so a long-running command such as a stalled
// file download stops instead of hanging, and exits non-zero on error.
func Main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "pfix:", err)
		os.Exit(1)
	}
}
