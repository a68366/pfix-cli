package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/config"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type loginOptions struct {
	profile    string
	in         io.Reader
	out        io.Writer
	readSecret func(prompt string) (string, error)
	validate   func(ctx context.Context, domain, token string) error
	configPath func() (string, error)
}

func newLoginCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to a Planfix account and save credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runLogin(cmd.Context(), loginOptions{
				profile:    firstNonEmpty(g.Profile, "default"),
				in:         cmd.InOrStdin(),
				out:        cmd.OutOrStdout(),
				readSecret: readSecretFromTerminal,
				validate:   validateCredentials,
				configPath: func() (string, error) { return config.DefaultPath(os.Getenv) },
			})
		},
	}
}

func runLogin(ctx context.Context, o loginOptions) error {
	reader := bufio.NewReader(o.in)

	fmt.Fprint(o.out, "Planfix domain (e.g. example.planfix.com): ")
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return fmt.Errorf("read domain: %w", err)
	}
	domain := strings.TrimSpace(line)
	if domain == "" {
		return fmt.Errorf("domain must not be empty")
	}

	token, err := o.readSecret("API token: ")
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token must not be empty")
	}

	if err := o.validate(ctx, domain, token); err != nil {
		return fmt.Errorf("credential check failed: %w", err)
	}

	path, err := o.configPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	cfg.Profiles[o.profile] = config.Profile{Domain: domain, Token: token}
	if cfg.CurrentProfile == "" {
		cfg.CurrentProfile = o.profile
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(o.out, "Saved profile %q for %s\n", o.profile, domain)
	return nil
}

func readSecretFromTerminal(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func validateCredentials(ctx context.Context, domain, token string) error {
	resp, err := planfix.New(domain, token).Do(ctx, "POST", "task/list", []byte(`{"pageSize":1}`), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}
	data, _ := io.ReadAll(resp.Body)
	return planfix.ParseError(resp.StatusCode, data)
}
