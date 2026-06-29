package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/buildinfo"
)

func TestVersionCommandPrintsVersion(t *testing.T) {
	c := newVersionCmd()
	buf := &bytes.Buffer{}
	c.SetOut(buf)

	if err := c.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(buf.String(), buildinfo.Version) {
		t.Errorf("output %q does not contain version %q", buf.String(), buildinfo.Version)
	}
}
