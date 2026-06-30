package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pfconfig "github.com/a68366/pfix-cli/internal/config"
)

func writeConfig(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const sampleCfg = `current_profile: staging
profiles:
  default:
    domain: a.planfix.com
    token: tokenAAAA
  staging:
    domain: b.planfix.com
    token: tokenBBBB
`

func TestRunList(t *testing.T) {
	t.Run("two profiles with active marker", func(t *testing.T) {
		path := writeConfig(t, sampleCfg)
		var buf bytes.Buffer
		if err := runList(path, &buf, false); err != nil {
			t.Fatalf("runList: %v", err)
		}
		got := buf.String()

		for _, want := range []string{"default", "staging", "a.planfix.com", "b.planfix.com"} {
			if !strings.Contains(got, want) {
				t.Errorf("output missing %q:\n%s", want, got)
			}
		}
		// active marker on the staging row
		lines := strings.Split(strings.TrimSpace(got), "\n")
		var stagingLine string
		for _, l := range lines {
			if strings.Contains(l, "staging") {
				stagingLine = l
				break
			}
		}
		if !strings.Contains(stagingLine, "*") {
			t.Errorf("staging row should contain '*', got: %q", stagingLine)
		}
		// default row must NOT have the marker
		var defaultLine string
		for _, l := range lines {
			if strings.Contains(l, "default") {
				defaultLine = l
				break
			}
		}
		if strings.Contains(defaultLine, "*") {
			t.Errorf("default row should not contain '*', got: %q", defaultLine)
		}
		// sorted: default before staging
		var di, si int
		for i, l := range lines {
			if strings.Contains(l, "default") {
				di = i
			}
			if strings.Contains(l, "staging") {
				si = i
			}
		}
		if di >= si {
			t.Errorf("expected default row before staging row (di=%d si=%d)", di, si)
		}
		// header present
		if !strings.Contains(got, "NAME") {
			t.Errorf("header 'NAME' missing in non-quiet output:\n%s", got)
		}
	})

	t.Run("quiet suppresses header", func(t *testing.T) {
		path := writeConfig(t, sampleCfg)
		var buf bytes.Buffer
		if err := runList(path, &buf, true); err != nil {
			t.Fatalf("runList quiet: %v", err)
		}
		got := buf.String()
		if strings.Contains(got, "NAME") {
			t.Errorf("header 'NAME' should be absent in quiet mode:\n%s", got)
		}
		// data still present
		if !strings.Contains(got, "default") || !strings.Contains(got, "staging") {
			t.Errorf("profile names missing in quiet output:\n%s", got)
		}
	})

	t.Run("empty config shows friendly message", func(t *testing.T) {
		path := writeConfig(t, `{}`)
		var buf bytes.Buffer
		if err := runList(path, &buf, false); err != nil {
			t.Fatalf("runList empty: %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, "No profiles configured") {
			t.Errorf("expected friendly message, got:\n%s", got)
		}
	})
}

func TestRunUse(t *testing.T) {
	t.Run("switch to default", func(t *testing.T) {
		path := writeConfig(t, sampleCfg)
		var buf bytes.Buffer
		if err := runUse(path, "default", &buf, false); err != nil {
			t.Fatalf("runUse: %v", err)
		}
		// re-load and verify
		cfg, err := pfconfig.Load(path)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if cfg.CurrentProfile != "default" {
			t.Errorf("CurrentProfile = %q, want %q", cfg.CurrentProfile, "default")
		}
		if !strings.Contains(buf.String(), "default") {
			t.Errorf("output should mention profile name, got: %q", buf.String())
		}
	})

	t.Run("unknown profile returns error", func(t *testing.T) {
		path := writeConfig(t, sampleCfg)
		var buf bytes.Buffer
		err := runUse(path, "nope", &buf, false)
		if err == nil {
			t.Fatal("expected error for unknown profile")
		}
		if !strings.Contains(err.Error(), "no such profile") {
			t.Errorf("error should mention 'no such profile', got: %v", err)
		}
	})
}

func TestRunShow(t *testing.T) {
	t.Run("staging profile masked token", func(t *testing.T) {
		path := writeConfig(t, sampleCfg)
		var buf bytes.Buffer
		if err := runShow(path, "staging", &buf); err != nil {
			t.Fatalf("runShow: %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, "b.planfix.com") {
			t.Errorf("output missing domain:\n%s", got)
		}
		if !strings.Contains(got, "****BBBB") {
			t.Errorf("output missing masked token '****BBBB':\n%s", got)
		}
		if strings.Contains(got, "tokenBBBB") {
			t.Errorf("output must NOT contain raw token:\n%s", got)
		}
	})

	t.Run("unknown profile returns error", func(t *testing.T) {
		path := writeConfig(t, sampleCfg)
		var buf bytes.Buffer
		err := runShow(path, "nope", &buf)
		if err == nil {
			t.Fatal("expected error for unknown profile")
		}
		if !strings.Contains(err.Error(), "no such profile") {
			t.Errorf("error should mention 'no such profile', got: %v", err)
		}
	})
}
