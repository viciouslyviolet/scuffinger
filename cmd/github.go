package cmd

import (
	"github.com/spf13/cobra"

	"scuffinger/internal/i18n"
)

func newGitHubCmd() *cobra.Command {
	gh := &cobra.Command{
		Use:         "github",
		Short:       i18n.Get(i18n.CmdGitHubShort),
		Long:        i18n.Get(i18n.CmdGitHubLong),
		Annotations: map[string]string{"skipBootstrap": "true"},
	}

	gh.AddCommand(newGitHubAuthCmd())
	gh.AddCommand(newGitHubStatusCmd())
	gh.AddCommand(newGitHubLogoutCmd())
	gh.AddCommand(newGitHubMonitorCmd())

	return gh
}
