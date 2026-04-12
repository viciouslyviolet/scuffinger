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
	"scuffinger/internal/services"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "serve",
		Short:       i18n.Get(i18n.CmdServeShort),
		Long:        i18n.Get(i18n.CmdServeLong),
		Annotations: map[string]string{"skipCollector": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Record uptime reference point
			metrics.RecordStartTime()
			cfg := GetConfig()
			mgr := GetManager()
			log := GetLogger()

			// ── Build optional route registrars ──────────────────────
			var registrars []server.RouteRegistrar

			if cfg.GitHub.Enabled() {
				ghSvc := mgr.ServiceByName("github")
				if gs, ok := ghSvc.(*services.GitHubService); ok {
					gh := server.NewGitHubHandler(gs, gs.Organization(), log)
					registrars = append(registrars, gh)
				}
			}

			// ── Auth endpoint (GitHub OAuth device flow) ────────────
			if clientID := cfg.GitHub.FirstClientID(); clientID != "" {
				authH := server.NewAuthHandler(clientID, log)
				registrars = append(registrars, authH)
			}

			// ── Debug browser (PostgreSQL + ValKey) ──────────────────
			dbSvc := mgr.ServiceByName("database")
			cacheSvc := mgr.ServiceByName("cache")
			if ds, ok := dbSvc.(*services.DatabaseService); ok {
				if cs, ok2 := cacheSvc.(*services.CacheService); ok2 {
					debug := server.NewDebugHandler(ds.Pool(), cs.Client(), log)
					registrars = append(registrars, debug)
				}
			}

			// ── HTTP server ──────────────────────────────────────────
			router := server.NewRouter(cfg, mgr, registrars...)

			srv := &http.Server{
				Addr:    cfg.Address(),
				Handler: router,
			}

			// Graceful shutdown
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				log.Info(i18n.Get(i18n.MsgServerStarting), "addr", cfg.Address())
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
