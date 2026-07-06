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
	flagProfile string
	force       bool
	env         func(string) string
	in          io.Reader
	out         io.Writer
	readSecret  func(prompt string) (string, error)
	validate    func(ctx context.Context, domain, token string) error
	configPath  func() (string, error)
}

func newLoginCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to a Planfix account and save credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runLogin(cmd.Context(), loginOptions{
				flagProfile: g.Profile,
				force:       force,
				env:         os.Getenv,
				in:          cmd.InOrStdin(),
				out:         cmd.OutOrStdout(),
				readSecret:  readSecretFromTerminal,
				validate:    validateCredentials,
				configPath:  func() (string, error) { return config.DefaultPath(os.Getenv) },
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing profile without confirmation")
	return cmd
}

func runLogin(ctx context.Context, o loginOptions) error {
	reader := bufio.NewReader(o.in)

	path, err := o.configPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	profile := config.ResolveProfileName(o.flagProfile, o.env, cfg)

	if existing, ok := cfg.Profiles[profile]; ok && !o.force {
		if !confirmOverwrite(reader, o.out, profile, existing.Domain) {
			fmt.Fprintf(o.out, "Canceled; profile %q left unchanged.\n", profile)
			return nil
		}
	}

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

	cfg.Profiles[profile] = config.Profile{Domain: domain, Token: token}
	if cfg.CurrentProfile == "" {
		cfg.CurrentProfile = profile
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Fprintf(o.out, "Saved profile %q for %s\n", profile, domain)
	return nil
}

// confirmOverwrite prints an actionable prompt naming the existing profile and
// its domain, points the user at --profile to keep both accounts, and returns
// true only when the answer is an affirmative y/yes. EOF or empty input declines.
func confirmOverwrite(reader *bufio.Reader, out io.Writer, name, domain string) bool {
	fmt.Fprintf(out, "Profile %q already exists (%s).\n", name, domain)
	fmt.Fprint(out, "To log in as a separate account, answer n and re-run with --profile <name>.\n")
	fmt.Fprintf(out, "Overwrite %q? [y/N]: ", name)
	line, _ := reader.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
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
