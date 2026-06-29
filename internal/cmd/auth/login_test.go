package auth

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/config"
)

func baseOpts(t *testing.T, validate func(context.Context, string, string) error) loginOptions {
	path := filepath.Join(t.TempDir(), "config.yml")
	return loginOptions{
		profile:    "default",
		in:         strings.NewReader("example.planfix.com\n"),
		out:        &strings.Builder{},
		readSecret: func(string) (string, error) { return "tok-123", nil },
		validate:   validate,
		configPath: func() (string, error) { return path, nil },
	}
}

func TestRunLoginSavesProfileOnSuccess(t *testing.T) {
	o := baseOpts(t, func(context.Context, string, string) error { return nil })
	if err := runLogin(context.Background(), o); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	path, _ := o.configPath()
	cfg, _ := config.Load(path)
	p := cfg.Profiles["default"]
	if p.Domain != "example.planfix.com" || p.Token != "tok-123" {
		t.Errorf("saved profile = %+v", p)
	}
	if cfg.CurrentProfile != "default" {
		t.Errorf("current profile = %q, want default", cfg.CurrentProfile)
	}
}

func TestRunLoginDoesNotSaveOnValidationFailure(t *testing.T) {
	o := baseOpts(t, func(context.Context, string, string) error { return errors.New("401") })
	if err := runLogin(context.Background(), o); err == nil {
		t.Fatal("expected error")
	}
	path, _ := o.configPath()
	cfg, _ := config.Load(path)
	if len(cfg.Profiles) != 0 {
		t.Errorf("profile should not be saved, got %v", cfg.Profiles)
	}
}

func TestMaskToken(t *testing.T) {
	if got := maskToken("abcdef"); got != "****cdef" {
		t.Errorf("maskToken = %q", got)
	}
	if got := maskToken("ab"); got != "****" {
		t.Errorf("maskToken short = %q", got)
	}
}
