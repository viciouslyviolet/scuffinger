package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"scuffinger/internal/auth"
	"scuffinger/internal/i18n"
	"scuffinger/internal/vault"
)

func newGitHubLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "logout",
		Short:       i18n.Get(i18n.CmdGitHubLogoutShort),
		Long:        i18n.Get(i18n.CmdGitHubLogoutLong),
		Annotations: map[string]string{"skipBootstrap": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			store := vault.New()
			auth.ClearCredentials(store)
			fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s\n", i18n.Get(i18n.MsgAuthLoggedOut))
			return nil
		},
	}
}
