package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolvePrecedence(t *testing.T) {
	cfg := &Config{
		CurrentProfile: "default",
		Profiles: map[string]Profile{
			"default": {Domain: "file.planfix.com", Token: "file-token"},
			"other":   {Domain: "other.planfix.com", Token: "other-token"},
		},
	}
	cases := []struct {
		name       string
		ov         Overrides
		env        map[string]string
		wantDomain string
		wantToken  string
	}{
		{"file only", Overrides{}, nil, "file.planfix.com", "file-token"},
		{"env beats file", Overrides{}, map[string]string{"PFIX_DOMAIN": "env.planfix.com", "PFIX_TOKEN": "env-token"}, "env.planfix.com", "env-token"},
		{"flag beats env", Overrides{Domain: "flag.planfix.com"}, map[string]string{"PFIX_DOMAIN": "env.planfix.com"}, "flag.planfix.com", "file-token"},
		{"env profile selects other", Overrides{}, map[string]string{"PFIX_PROFILE": "other"}, "other.planfix.com", "other-token"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := Resolve(cfg, c.ov, envFrom(c.env))
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if res.Domain != c.wantDomain || res.Token != c.wantToken {
				t.Errorf("got (%q,%q), want (%q,%q)", res.Domain, res.Token, c.wantDomain, c.wantToken)
			}
		})
	}
}

func TestResolveMissingErrors(t *testing.T) {
	_, err := Resolve(&Config{Profiles: map[string]Profile{}}, Overrides{}, envFrom(nil))
	if err == nil {
		t.Fatal("expected error when no credentials are available")
	}
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("got %v, want ErrNotAuthenticated", err)
	}
}

func TestDefaultPath(t *testing.T) {
	want := filepath.Join("custom", "path", "config.yml")
	got, err := DefaultPath(envFrom(map[string]string{"PFIX_CONFIG": want}))
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got, err = DefaultPath(envFrom(nil))
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	suffix := filepath.Join("pfix", "config.yml")
	if got == "" || !strings.HasSuffix(got, suffix) {
		t.Errorf("got %q, want non-empty path ending in %q", got, suffix)
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %v", cfg.Profiles)
	}
}

func TestSaveLoadRoundTripAndMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.yml")
	in := &Config{CurrentProfile: "default", Profiles: map[string]Profile{"default": {Domain: "d", Token: "t"}}}
	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Profiles["default"].Token != "t" {
		t.Errorf("round-trip lost token: %+v", out)
	}
}
