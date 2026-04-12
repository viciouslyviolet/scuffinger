package services

import (
	"context"
	"time"

	dbgithub "scuffinger/database/github"
	"scuffinger/internal/auth"
	"scuffinger/internal/config"
	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
	"scuffinger/internal/vault"
)

// BootstrapOpts controls startup behaviour.
type BootstrapOpts struct {
	// HealthCheckInterval is how often periodic pings run. Zero disables them.
	HealthCheckInterval time.Duration

	// SkipCollector disables the GitHub collector loop. Set to true for API-only
	// mode (the "serve" command) so that collection is handled by a separate process.
	SkipCollector bool
}

// DefaultBootstrapOpts returns sensible defaults.
func DefaultBootstrapOpts() BootstrapOpts {
	return BootstrapOpts{
		HealthCheckInterval: 10 * time.Second,
	}
}

// Bootstrap creates every registered service, connects to them, runs
// self-tests, and starts periodic health checks.
// The caller MUST call Manager.CloseAll when done.
//
// To add a new service, append it to the services slice below.
func Bootstrap(ctx context.Context, cfg *config.Config, opts BootstrapOpts, log *logging.Logger) (*Manager, error) {
	// ── Register services (add new ones here) ────────────────────────
	cacheSvc := NewCacheService(cfg.Cache, log)
	dbSvc := NewDatabaseService(cfg.Database, log)
	svcs := []Service{
		cacheSvc,
		dbSvc,
	}

	// Optional services
	var ghSvc *GitHubService

	// If no tokens in config, try the system vault (populated by 'github auth')
	if len(cfg.GitHub.Tokens) == 0 {
		store := vault.New()
		if tok, err := auth.LoadToken(store); err == nil && tok != "" {
			log.Info(i18n.Get(i18n.MsgAuthTokenFromVault))
			cfg.GitHub.Tokens = append(cfg.GitHub.Tokens, tok)
		}
	}

	if cfg.GitHub.Enabled() {
		ghSvc = NewGitHubService(cfg.GitHub, log)
		svcs = append(svcs, ghSvc)
	}

	mgr := NewManager(log, svcs...)

	// ── Connect ──────────────────────────────────────────────────────
	log.Info(i18n.Get(i18n.MsgBootstrapConnecting))
	if err := mgr.ConnectAll(ctx); err != nil {
		return nil, i18n.Err(i18n.ErrBootstrapConnect, err)
	}

	// ── Create GitHub cache tables (idempotent) ─────────────────────
	if _, err := dbSvc.Pool().Exec(ctx, dbgithub.CreateTables); err != nil {
		log.Warn("Failed to create GitHub tables", "error", err)
	}

	// ── GitHub collector (needs a connected client) ──────────────────
	var collector *GitHubCollectorService
	if !opts.SkipCollector && ghSvc != nil && len(cfg.GitHub.Repositories) > 0 {
		collector = NewGitHubCollectorService(cfg.GitHub, ghSvc, dbSvc.Pool(), cacheSvc.Client(), log)
		mgr.AddService(collector)
		// Connect the collector (no-op but fulfils the contract)
		if err := collector.Connect(ctx); err != nil {
			return nil, i18n.Err(i18n.ErrBootstrapConnect, err)
		}
	}

	// ── Self-tests ───────────────────────────────────────────────────
	log.Info(i18n.Get(i18n.MsgBootstrapSelfTests))
	if err := mgr.RunSelfTests(ctx); err != nil {
		_ = mgr.CloseAll()
		return nil, i18n.Err(i18n.ErrBootstrapSelfTests, err)
	}
	log.Info(i18n.Get(i18n.MsgBootstrapTestsPassed))

	// ── Periodic health checks ───────────────────────────────────────
	if opts.HealthCheckInterval > 0 {
		mgr.StartHealthChecks(opts.HealthCheckInterval)
		log.Debug(i18n.Get(i18n.MsgBootstrapHealthStart), "interval", opts.HealthCheckInterval.String())
	}

	// ── Start background collector ───────────────────────────────────
	if collector != nil {
		collector.Start()
	}

	return mgr, nil
}
