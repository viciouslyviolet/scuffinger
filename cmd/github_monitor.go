package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"scuffinger/internal/i18n"
	"scuffinger/internal/metrics"
	"scuffinger/internal/server"
)

func newGitHubMonitorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "monitor",
		Short: i18n.Get(i18n.CmdGitHubMonitorShort),
		Long:  i18n.Get(i18n.CmdGitHubMonitorLong),
		RunE: func(cmd *cobra.Command, args []string) error {
			metrics.RecordStartTime()
			cfg := GetConfig()
			mgr := GetManager()
			log := GetLogger()

			// Minimal router: health + metrics only (no API proxy routes).
			router := server.NewRouter(cfg, mgr)

			srv := &http.Server{
				Addr:    cfg.Address(),
				Handler: router,
			}

			// Graceful shutdown
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				log.Info(i18n.Get(i18n.MsgServerStarting), "addr", cfg.Address(), "mode", "collector")
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error(i18n.Get(i18n.ErrServerListen), "error", err)
				}
			}()

			<-quit
			log.Info(i18n.Get(i18n.MsgServerShutdown))

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				return i18n.Err(i18n.ErrServerShutdown, err)
			}

			log.Info(i18n.Get(i18n.MsgServerStopped))
			return nil
		},
	}
}

