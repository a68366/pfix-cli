package cmd

import "testing"

func TestNewRootCmdRegistersJQFlag(t *testing.T) {
	root := NewRootCmd()
	f := root.PersistentFlags().Lookup("jq")
	if f == nil {
		t.Fatal("expected root command to register a persistent --jq flag")
	}
}
