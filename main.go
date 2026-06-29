package main

import (
	"fmt"
	"os"

	"github.com/a68366/pfix-cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "pfix:", err)
		os.Exit(1)
	}
}
