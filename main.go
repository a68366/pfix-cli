// Command pfix-cli is a compatibility entry point at the module root. It keeps
// `go install github.com/a68366/pfix-cli@latest` working, though that names
// the binary pfix-cli; the canonical entry point is ./cmd/pfix, which installs
// as pfix.
package main

import "github.com/a68366/pfix-cli/internal/cmd"

func main() {
	cmd.Main()
}
