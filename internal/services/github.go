package services

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"

	"scuffinger/internal/config"
	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
	"scuffinger/internal/metrics"
)

// clientEntry holds a single GitHub API credential and its rate-limit state.
type clientEntry struct {
	client    *github.Client
	label     string    // e.g. "pat-0", "app-0"
	remaining int       // last-known remaining rate limit
	resetAt   time.Time // when the rate limit resets
}

// GitHubService manages a pool of GitHub API clients and transparently
// rotates to the next credential when the active one's rate limit is low.
type GitHubService struct {
	cfg     config.GitHubConfig
	entries []*clientEntry
	active  int
	mu      sync.RWMutex
	log     *logging.Logger
}

// NewGitHubService creates a new GitHubService.
func NewGitHubService(cfg config.GitHubConfig, log *logging.Logger) *GitHubService {
	return &GitHubService{cfg: cfg, log: log}
}

func (s *GitHubService) Name() string { return "github" }

func (s *GitHubService) Connect(ctx context.Context) error {
	if !s.cfg.Enabled() {
		return errors.New(i18n.Get(i18n.ErrGhNotConfigured))
	}

	// ── Build PAT clients ────────────────────────────────────────────
	for i, token := range s.cfg.Tokens {
		if token == "" {
			continue
		}
		label := fmt.Sprintf("pat-%d", i)
		s.log.Debug(i18n.Get(i18n.MsgGhAuthToken), "credential", label)
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		c := github.NewClient(oauth2.NewClient(ctx, ts))
		s.entries = append(s.entries, &clientEntry{client: c, label: label, remaining: -1})
	}

	// ── Build App clients ────────────────────────────────────────────
	for i, app := range s.cfg.Applications {
		if app.AppID == 0 {
			continue
		}
		label := fmt.Sprintf("app-%d", i)
		s.log.Debug(i18n.Get(i18n.MsgGhAuthApp),
			"credential", label,
			"app_id", app.AppID,
			"installation_id", app.InstallationID,
		)

		keyData, err := os.ReadFile(app.PrivateKeyPath)
		if err != nil {
			return i18n.Err(i18n.ErrGhReadKey, err)
		}

		block, _ := pem.Decode(keyData)
		if block == nil {
			return fmt.Errorf("%s: PEM block not found (%s)", i18n.Get(i18n.ErrGhParseKey), label)
		}

		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			pk8, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err2 != nil {
				return i18n.Err(i18n.ErrGhParseKey, err)
			}
			var ok bool
			key, ok = pk8.(*rsa.PrivateKey)
			if !ok {
				return fmt.Errorf("%s: not an RSA key (%s)", i18n.Get(i18n.ErrGhParseKey), label)
			}
		}

		transport := newGHAppTransport(app.AppID, app.InstallationID, key)
		c := github.NewClient(&http.Client{Transport: transport})
		s.entries = append(s.entries, &clientEntry{client: c, label: label, remaining: -1})
	}

	if len(s.entries) == 0 {
		return errors.New(i18n.Get(i18n.ErrGhNotConfigured))
	}

	s.log.Info("GitHub client pool initialized", "credentials", len(s.entries))
	return nil
}

// SelfTest verifies authentication and checks the rate limit for the active client.
func (s *GitHubService) SelfTest(ctx context.Context) error {
	e := s.activeEntry()

	// Verify authentication
	_, _, err := e.client.Users.Get(ctx, "")
	if err != nil {
		return i18n.Err(i18n.ErrGhAuth, err)
	}

	// Check rate limit
	limits, _, err := e.client.RateLimit.Get(ctx)
	if err != nil {
		return i18n.Err(i18n.ErrGhRateLimit, err)
	}

	core := limits.Core
	s.mu.Lock()
	e.remaining = core.Remaining
	e.resetAt = core.Reset.Time
	s.mu.Unlock()

	metrics.SetGitHubRateLimit(e.label, core.Remaining)
	s.log.Info(i18n.Get(i18n.MsgGhRateRemaining),
		"credential", e.label,
		"remaining", core.Remaining,
		"limit", core.Limit,
		"resets_at", core.Reset.Time.String(),
	)

	threshold := s.cfg.RateLimitThreshold
	if threshold == 0 {
		threshold = 100
	}
	if core.Remaining < threshold {
		s.log.Warn(i18n.Get(i18n.WarnGhRateLow),
			"credential", e.label,
			"remaining", core.Remaining,
			"threshold", threshold,
		)
		s.rotate(ctx)
	}

	s.log.Info(i18n.Get(i18n.MsgGhSelfTestPassed))
	return nil
}

// Ping checks the rate limit and rotates if needed.
func (s *GitHubService) Ping(ctx context.Context) error {
	e := s.activeEntry()

	limits, _, err := e.client.RateLimit.Get(ctx)
	if err != nil {
		return i18n.Err(i18n.ErrGhFetchRateLimit, err)
	}

	s.mu.Lock()
	e.remaining = limits.Core.Remaining
	e.resetAt = limits.Core.Reset.Time
	s.mu.Unlock()

	metrics.SetGitHubRateLimit(e.label, limits.Core.Remaining)

	threshold := s.cfg.RateLimitThreshold
	if threshold == 0 {
		threshold = 100
	}

	if limits.Core.Remaining < threshold {
		s.log.Warn("Rate limit low, rotating credential",
			"credential", e.label,
			"remaining", limits.Core.Remaining,
			"threshold", threshold,
		)
		s.rotate(ctx)
	}
	return nil
}

func (s *GitHubService) Close() error { return nil }

// Client returns the currently active GitHub API client.
// Callers should NOT cache this — call Client() on each use to benefit from rotation.
func (s *GitHubService) Client() *github.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[s.active].client
}

// ActiveLabel returns the label of the currently active credential.
func (s *GitHubService) ActiveLabel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[s.active].label
}

// Organization returns the configured organization, if any.
func (s *GitHubService) Organization() string {
	return s.cfg.Organization
}

// CredentialCount returns the total number of configured credentials.
func (s *GitHubService) CredentialCount() int {
	return len(s.entries)
}

// ── internal ─────────────────────────────────────────────────────────────────

// activeEntry returns the current active entry (read-locked).
func (s *GitHubService) activeEntry() *clientEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[s.active]
}

// rotate tries to switch to the next credential with remaining rate limit
// above the threshold. If all are exhausted, stays on the one with the
// highest remaining count.
func (s *GitHubService) rotate(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.entries) <= 1 {
		return // nothing to rotate to
	}

	threshold := s.cfg.RateLimitThreshold
	if threshold == 0 {
		threshold = 100
	}

	best := s.active
	bestRemaining := s.entries[s.active].remaining

	for i := 1; i < len(s.entries); i++ {
		idx := (s.active + i) % len(s.entries)
		e := s.entries[idx]

		// Fetch fresh rate limit for this candidate (unlocked briefly)
		s.mu.Unlock()
		limits, _, err := e.client.RateLimit.Get(ctx)
		s.mu.Lock()

		if err != nil {
			continue
		}
		e.remaining = limits.Core.Remaining
		e.resetAt = limits.Core.Reset.Time
		metrics.SetGitHubRateLimit(e.label, limits.Core.Remaining)

		if e.remaining >= threshold {
			s.active = idx
			s.log.Info("Rotated to new GitHub credential",
				"credential", e.label,
				"remaining", e.remaining,
			)
			return
		}
		if e.remaining > bestRemaining {
			best = idx
			bestRemaining = e.remaining
		}
	}

	// All exhausted — pick the one with the most headroom
	if best != s.active {
		s.active = best
		s.log.Warn("All GitHub credentials near rate limit, using best available",
			"credential", s.entries[best].label,
			"remaining", bestRemaining,
		)
	}
}
