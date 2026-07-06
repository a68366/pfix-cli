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

func seedProfile(t *testing.T, path, name, domain, token string) {
	t.Helper()
	cfg := &config.Config{
		CurrentProfile: name,
		Profiles:       map[string]config.Profile{name: {Domain: domain, Token: token}},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

func TestRunLoginConfirmsBeforeOverwritingExistingProfile(t *testing.T) {
	o := baseOpts(t, func(context.Context, string, string) error { return nil })
	path, _ := o.configPath()
	seedProfile(t, path, "default", "old.planfix.com", "old-tok")
	o.in = strings.NewReader("y\nnew.planfix.com\n")
	o.readSecret = func(string) (string, error) { return "new-tok", nil }

	if err := runLogin(context.Background(), o); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	cfg, _ := config.Load(path)
	if p := cfg.Profiles["default"]; p.Domain != "new.planfix.com" || p.Token != "new-tok" {
		t.Errorf("profile after confirmed overwrite = %+v", p)
	}
}

func TestRunLoginAbortsOverwriteWhenDeclined(t *testing.T) {
	o := baseOpts(t, func(context.Context, string, string) error { return nil })
	path, _ := o.configPath()
	seedProfile(t, path, "default", "old.planfix.com", "old-tok")
	o.in = strings.NewReader("n\n")

	if err := runLogin(context.Background(), o); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	cfg, _ := config.Load(path)
	if p := cfg.Profiles["default"]; p.Domain != "old.planfix.com" || p.Token != "old-tok" {
		t.Errorf("declined overwrite changed profile: %+v", p)
	}
}

func TestRunLoginForceSkipsOverwriteConfirmation(t *testing.T) {
	o := baseOpts(t, func(context.Context, string, string) error { return nil })
	o.force = true
	path, _ := o.configPath()
	seedProfile(t, path, "default", "old.planfix.com", "old-tok")
	o.in = strings.NewReader("new.planfix.com\n")
	o.readSecret = func(string) (string, error) { return "new-tok", nil }

	if err := runLogin(context.Background(), o); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	cfg, _ := config.Load(path)
	if p := cfg.Profiles["default"]; p.Domain != "new.planfix.com" || p.Token != "new-tok" {
		t.Errorf("force overwrite = %+v", p)
	}
}

func TestRunLoginOverwritePromptIsActionable(t *testing.T) {
	o := baseOpts(t, func(context.Context, string, string) error { return nil })
	path, _ := o.configPath()
	seedProfile(t, path, "default", "old.planfix.com", "old-tok")
	out := &strings.Builder{}
	o.out = out
	o.in = strings.NewReader("n\n")

	if err := runLogin(context.Background(), o); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "--profile") {
		t.Errorf("prompt should mention --profile, got: %q", got)
	}
	if !strings.Contains(got, "default") {
		t.Errorf("prompt should name the existing profile, got: %q", got)
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
