package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"scuffinger/internal/auth"
	"scuffinger/internal/i18n"
	"scuffinger/internal/vault"
)

func newGitHubStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "status",
		Short:       i18n.Get(i18n.CmdGitHubStatusShort),
		Long:        i18n.Get(i18n.CmdGitHubStatusLong),
		Annotations: map[string]string{"skipBootstrap": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			store := vault.New()
			cfg := GetConfig()

			// Check config-based auth
			if len(cfg.GitHub.Tokens) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %d token(s) configured in config file / env\n", len(cfg.GitHub.Tokens))
				return nil
			}
			if len(cfg.GitHub.Applications) > 0 {
				for i, app := range cfg.GitHub.Applications {
					fmt.Fprintf(cmd.OutOrStdout(), "  ✓ GitHub App %d configured (app_id: %d)\n", i, app.AppID)
				}
				return nil
			}

			// Check vault
			if !auth.HasCredentials(store) {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✗ %s\n", i18n.Get(i18n.MsgAuthStatusNoToken))
				return nil
			}

			username, _ := auth.LoadUser(store)
			if username != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s as %s (token in system vault)\n", i18n.Get(i18n.MsgAuthStatusLoggedIn), username)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s (token in system vault)\n", i18n.Get(i18n.MsgAuthStatusLoggedIn))
			}

			return nil
		},
	}
}
