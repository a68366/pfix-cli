package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile holds credentials for one Planfix account.
type Profile struct {
	Domain string `yaml:"domain"`
	Token  string `yaml:"token"`
}

// Config is the on-disk configuration.
type Config struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

// DefaultPath returns the config file path, honoring PFIX_CONFIG.
func DefaultPath(env func(string) string) (string, error) {
	if p := env("PFIX_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate config dir: %w", err)
	}
	return filepath.Join(dir, "pfix", "config.yml"), nil
}

// Load reads the config file. A missing file yields an empty Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Profiles: map[string]Profile{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := &Config{Profiles: map[string]Profile{}}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

// Save writes the config file atomically with 0600 permissions.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// Overrides are flag-sourced values that take top precedence.
type Overrides struct {
	Profile string
	Domain  string
	Token   string
}

// Resolved is the effective credential set for a command.
type Resolved struct {
	ProfileName string
	Domain      string
	Token       string
}

// ErrNotAuthenticated indicates no usable domain/token was found.
var ErrNotAuthenticated = errors.New("not authenticated: run `pfix auth login` or set PFIX_DOMAIN and PFIX_TOKEN")

// Resolve applies precedence flags > env > config file.
func Resolve(cfg *Config, ov Overrides, env func(string) string) (Resolved, error) {
	name := firstNonEmpty(ov.Profile, env("PFIX_PROFILE"), cfg.CurrentProfile, "default")
	p := cfg.Profiles[name]

	res := Resolved{
		ProfileName: name,
		Domain:      firstNonEmpty(ov.Domain, env("PFIX_DOMAIN"), p.Domain),
		Token:       firstNonEmpty(ov.Token, env("PFIX_TOKEN"), p.Token),
	}
	if res.Domain == "" || res.Token == "" {
		return res, ErrNotAuthenticated
	}
	return res, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
