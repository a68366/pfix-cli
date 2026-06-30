package auth

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/config"
)

func baseOpts(t *testing.T, validate func(context.Context, string, string) error) loginOptions {
	path := filepath.Join(t.TempDir(), "config.yml")
	return loginOptions{
		flagProfile: "default",
		env:         func(string) string { return "" },
		in:          strings.NewReader("example.planfix.com\n"),
		out:         &strings.Builder{},
		readSecret:  func(string) (string, error) { return "tok-123", nil },
		validate:    validate,
		configPath:  func() (string, error) { return path, nil },
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

func TestRunLoginRespectsPFIXPROFILE(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	o := loginOptions{
		flagProfile: "",
		env: func(k string) string {
			if k == "PFIX_PROFILE" {
				return "work"
			}
			return ""
		},
		in:         strings.NewReader("example.planfix.com\n"),
		out:        &strings.Builder{},
		readSecret: func(string) (string, error) { return "tok-123", nil },
		validate:   func(context.Context, string, string) error { return nil },
		configPath: func() (string, error) { return path, nil },
	}
	if err := runLogin(context.Background(), o); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	cfg, _ := config.Load(path)
	if _, ok := cfg.Profiles["work"]; !ok {
		t.Errorf("expected credentials in profile 'work', got profiles: %v", cfg.Profiles)
	}
	if _, ok := cfg.Profiles["default"]; ok {
		t.Errorf("should not have written to 'default' profile, got: %v", cfg.Profiles)
	}
}

func TestMaskToken(t *testing.T) {
	if got := cmdutil.MaskToken("abcdef"); got != "****cdef" {
		t.Errorf("MaskToken = %q", got)
	}
	if got := cmdutil.MaskToken("ab"); got != "****" {
		t.Errorf("MaskToken short = %q", got)
	}
}
