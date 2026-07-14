package cmd

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewRootCmdRegistersJQFlag(t *testing.T) {
	root := NewRootCmd()
	f := root.PersistentFlags().Lookup("jq")
	if f == nil {
		t.Fatal("expected root command to register a persistent --jq flag")
	}
}

// newProbeRoot returns the real root command (with its PersistentPreRunE
// wiring) plus a test-only "probe" subcommand that records whether its RunE
// ran. The probe stands in for a real API-calling command (ping, task list, …)
// but touches neither config nor the network, so the tests below stay
// hermetic and deterministic in CI.
func newProbeRoot(ran *bool) *cobra.Command {
	root := NewRootCmd()
	root.AddCommand(&cobra.Command{
		Use:  "probe",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			*ran = true
			return nil
		},
	})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root
}

// TestPersistentPreRunEFailsFastOnInvalidJQ proves the root command's
// PersistentPreRunE wiring runs GlobalOpts.PreRun and rejects an invalid --jq
// expression before any subcommand's RunE executes — i.e. before a client is
// built or the network is reached. The probe subcommand inherits root's
// PersistentPreRunE (cobra runs the closest hook), so its RunE must stay
// untouched on the fail-fast path.
func TestPersistentPreRunEFailsFastOnInvalidJQ(t *testing.T) {
	var ranRunE bool
	root := newProbeRoot(&ranRunE)
	root.SetArgs([]string{"probe", "--jq", "("})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error from the invalid --jq expression")
	}
	// Assert the specific compile error, not merely that some error occurred, so
	// this distinguishes a PreRun short-circuit from a RunE/config/network
	// failure — only the former proves the wiring runs and fails fast.
	if !strings.Contains(err.Error(), "invalid --jq expression") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "invalid --jq expression")
	}
	if ranRunE {
		t.Fatal("subcommand RunE ran; PreRun did not short-circuit before the command executed")
	}
}

// TestPersistentPreRunERunsRunEForValidJQ is the control for the fail-fast
// test: with a valid --jq the same probe subcommand's RunE does run. Without
// this, the fail-fast assertion could pass vacuously (e.g. if the probe never
// ran for an unrelated reason).
func TestPersistentPreRunERunsRunEForValidJQ(t *testing.T) {
	var ranRunE bool
	root := newProbeRoot(&ranRunE)
	root.SetArgs([]string{"probe", "--jq", ".task.id"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error with a valid --jq: %v", err)
	}
	if !ranRunE {
		t.Fatal("probe RunE did not run with a valid --jq; the fail-fast assertion would be vacuous")
	}
}

func TestExecuteAcceptsContext(t *testing.T) {
	// Compiles only if Execute takes a context; a cancelled context still lets
	// a no-op command (version) return without hanging.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = ctx // Execute builds its own tree; this asserts the signature/handoff.
	if err := Execute(ctx); err != nil {
		t.Fatalf("Execute(ctx) with no args: %v", err)
	}
}
