package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"scuffinger/internal/config"
	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
	"scuffinger/internal/services"
)

var (
	cfgFile string
	appCfg  *config.Config
	appMgr  *services.Manager
	appLog  *logging.Logger
)

// NewRootCommand creates and returns the root cobra command with all subcommands registered.
// This is useful for testing.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "scuffinger",
		Short: i18n.Get(i18n.CmdRootShort),
		Long:  i18n.Get(i18n.CmdRootLong),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// ── Load configuration ───────────────────────────────
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.Get(i18n.ErrConfigLoad), err)
			}
			appCfg = cfg

			// ── Create logger ────────────────────────────────────
			appLog = logging.New(cfg.Log)
			appLog.Debug(i18n.Get(i18n.MsgConfigLoaded), "file", cfgFile, "level", cfg.Log.Level, "format", cfg.Log.Format)

			// ── Bootstrap services (unless the command opts out) ──
			if !needsBootstrap(cmd) {
				return nil
			}

			ctx := context.Background()
			opts := services.DefaultBootstrapOpts()
			if cmd.Annotations["skipCollector"] == "true" {
				opts.SkipCollector = true
			}
			mgr, err := services.Bootstrap(ctx, cfg, opts, appLog)
			if err != nil {
				return err
			}
			appMgr = mgr
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if appMgr != nil {
				return appMgr.CloseAll()
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "config/config.yaml", i18n.Get(i18n.CmdFlagConfig))

	// Add subcommands
	root.AddCommand(newVersionCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newGitHubCmd())

	return root
}

// needsBootstrap returns false for commands that should skip service initialisation.
func needsBootstrap(cmd *cobra.Command) bool {
	// Explicit opt-out via annotation
	if cmd.Annotations["skipBootstrap"] == "true" {
		return false
	}
	// Root by itself just shows help
	if !cmd.HasParent() {
		return false
	}
	return true
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = NewRootCommand()

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// GetConfig returns the loaded application configuration.
func GetConfig() *config.Config {
	return appCfg
}

// GetManager returns the bootstrapped service manager.
func GetManager() *services.Manager {
	return appMgr
}

// GetLogger returns the application logger.
func GetLogger() *logging.Logger {
	return appLog
}
