package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"scuffinger/internal/i18n"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "version",
		Short:       i18n.Get(i18n.CmdVersionShort),
		Long:        i18n.Get(i18n.CmdVersionLong),
		Annotations: map[string]string{"skipBootstrap": "true"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "scuffinger version %s\n", Version)
		},
	}
}
