package auth

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/config"
)

func TestRunLogoutRespectsPFIXPROFILE(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	initial := &config.Config{
		CurrentProfile: "work",
		Profiles: map[string]config.Profile{
			"work":    {Domain: "work.planfix.com", Token: "tok-work"},
			"default": {Domain: "def.planfix.com", Token: "tok-def"},
		},
	}
	if err := config.Save(path, initial); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Simulate: PFIX_PROFILE=work, no flag → resolves to "work"
	env := func(k string) string {
		if k == "PFIX_PROFILE" {
			return "work"
		}
		return ""
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	name := config.ResolveProfileName("", env, cfg)
	out := &strings.Builder{}
	if err := runLogout(path, name, out); err != nil {
		t.Fatalf("runLogout: %v", err)
	}

	after, _ := config.Load(path)
	if _, ok := after.Profiles["work"]; ok {
		t.Errorf("expected 'work' profile to be removed, but it still exists")
	}
	if _, ok := after.Profiles["default"]; !ok {
		t.Errorf("'default' profile should be untouched")
	}
}
