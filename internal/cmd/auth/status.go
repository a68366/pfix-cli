package auth

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

func newStatusCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the active profile and check the token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, res, err := g.Client()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Profile: %s\n", res.ProfileName)
			fmt.Fprintf(out, "Domain:  %s\n", res.Domain)
			fmt.Fprintf(out, "Token:   %s\n", maskToken(res.Token))

			resp, err := client.Do(cmd.Context(), "POST", "task/list", []byte(`{"pageSize":1}`), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)
			if resp.StatusCode == 200 {
				fmt.Fprintln(out, "Status:  valid")
				return nil
			}
			fmt.Fprintf(out, "Status:  invalid (HTTP %d)\n", resp.StatusCode)
			return fmt.Errorf("token check failed (HTTP %d)", resp.StatusCode)
		},
	}
}

func maskToken(t string) string {
	if len(t) <= 4 {
		return "****"
	}
	return "****" + t[len(t)-4:]
}
