package buildinfo

import "fmt"

// Populated at build time via -ldflags -X.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a one-line human-readable version summary.
func String() string {
	return fmt.Sprintf("pfix %s (commit %s, built %s)", Version, Commit, Date)
}
