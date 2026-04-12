package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	dbgithub "scuffinger/database/github"
	"scuffinger/internal/config"
	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
	"scuffinger/internal/metrics"
)

// GitHubCollectorService periodically fetches repository metadata,
// workflow runs, and job step timings from GitHub and exports them
// as Prometheus metrics.
//
// When multiple instances run in parallel (e.g. Kubernetes HPA), each
// repository is collected by at most one instance per cycle using a
// distributed lock in ValKey (Redis-compatible).
type GitHubCollectorService struct {
	cfg   config.GitHubConfig
	ghSvc *GitHubService
	pool  *pgxpool.Pool
	rdb   *redis.Client // nil = locking disabled (single instance)
	log   *logging.Logger

	podID   string // used as the lock value for debugging (pod name in K8s)
	cancel  context.CancelFunc
	lastErr error
}

// NewGitHubCollectorService creates a collector that uses the given GitHubService for client rotation.
// Pass a non-nil redis.Client to enable distributed locking for multi-instance deployments.
func NewGitHubCollectorService(cfg config.GitHubConfig, ghSvc *GitHubService, pool *pgxpool.Pool, rdb *redis.Client, log *logging.Logger) *GitHubCollectorService {
	podID, _ := os.Hostname()
	if podID == "" {
		podID = fmt.Sprintf("pod-%d", os.Getpid())
	}
	return &GitHubCollectorService{
		cfg:   cfg,
		ghSvc: ghSvc,
		pool:  pool,
		rdb:   rdb,
		log:   log,
		podID: podID,
	}
}

func (s *GitHubCollectorService) Name() string { return "github_collector" }

// Connect is a no-op — the collector reuses the already-connected GitHubService client.
func (s *GitHubCollectorService) Connect(_ context.Context) error { return nil }

// SelfTest validates the config and does one trial fetch on the first repo.
func (s *GitHubCollectorService) SelfTest(ctx context.Context) error {
	if len(s.cfg.Repositories) == 0 {
		return fmt.Errorf("%s", i18n.Get(i18n.ErrGhCollectorNoRepos))
	}

	owner, repo, err := parseRepo(s.cfg.Repositories[0])
	if err != nil {
		return err
	}

	_, _, err = s.ghSvc.Client().Repositories.Get(ctx, owner, repo)
	if err != nil {
		return i18n.Err(i18n.ErrGhCollectorFetchRepo, err)
	}

	s.log.Info(i18n.Get(i18n.MsgGhCollectorPassed), "repo", s.cfg.Repositories[0])
	return nil
}

// Ping returns the error from the last collection cycle (nil = healthy).
func (s *GitHubCollectorService) Ping(_ context.Context) error {
	return s.lastErr
}

// Close stops the background collection loop.
func (s *GitHubCollectorService) Close() error {
	if s.cancel != nil {
		s.cancel()
		s.log.Info(i18n.Get(i18n.MsgGhCollectorStopped))
	}
	return nil
}

// Start begins the periodic background collection loop.
// It must be called after Bootstrap (Connect + SelfTest).
func (s *GitHubCollectorService) Start() {
	interval := s.cfg.CollectorDuration()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.log.Info(i18n.Get(i18n.MsgGhCollectorStarting),
		"interval", interval.String(),
		"repos", len(s.cfg.Repositories),
	)

	go func() {
		// Run once immediately at startup
		s.collect(ctx)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.collect(ctx)
			}
		}
	}()
}

// collect runs a single collection cycle across all configured repositories.
// In multi-instance mode, a distributed lock is acquired per repo so that
// each repository is collected by exactly one instance per cycle.
func (s *GitHubCollectorService) collect(ctx context.Context) {
	s.log.Debug(i18n.Get(i18n.MsgGhCollectorTick))

	// Track which repos this instance actually collected so we only
	// reset stale metrics for those repos (not repos owned by other pods).
	collected := make(map[string]struct{})

	var lastErr error
	for _, fullName := range s.cfg.Repositories {
		owner, repo, err := parseRepo(fullName)
		if err != nil {
			s.log.Error(i18n.Get(i18n.ErrGhCollectorParseRepo), "repo", fullName, "error", err)
			lastErr = err
			continue
		}

		// ── Distributed lock ─────────────────────────────────────
		if !s.tryLock(ctx, fullName) {
			continue // another instance owns this repo
		}

		collected[fullName] = struct{}{}

		if err := s.collectRepo(ctx, owner, repo); err != nil {
			metrics.CollectorErrorsTotal.WithLabelValues(fullName).Inc()
			lastErr = err
			// Continue to the next repo — don't abort the cycle
		}
	}

	// Reset stale gauge series only for repos this pod collected.
	if len(collected) > 0 {
		metrics.ResetTrackedWorkflowMetricsForRepos(collected)
	}

	metrics.CollectorCyclesTotal.Inc()
	s.lastErr = lastErr
}

// tryLock attempts to acquire a distributed lock for the given repo.
// Returns true if this instance should collect the repo.
//
// Lock semantics:
//   - Key:  scuffinger:collect:<owner/repo>
//   - Value: pod identity (for debugging via valkey-cli)
//   - TTL:  collector interval (auto-expires; no explicit unlock needed)
//
// If ValKey is unavailable (rdb == nil or network error), the method
// returns true so single-instance and degraded-mode deployments still work.
func (s *GitHubCollectorService) tryLock(ctx context.Context, repo string) bool {
	if s.rdb == nil {
		return true // locking disabled — single-instance mode
	}

	lockKey := "scuffinger:collect:" + repo
	ttl := s.cfg.CollectorDuration()

	ok, err := s.rdb.SetNX(ctx, lockKey, s.podID, ttl).Result()
	if err != nil {
		// ValKey unreachable — degrade gracefully to non-locked mode
		s.log.Warn(i18n.Get(i18n.WarnGhCollectorLockError), "repo", repo, "error", err)
		return true
	}

	if !ok {
		// Another instance holds the lock — skip
		s.log.Debug(i18n.Get(i18n.MsgGhCollectorLockSkipped), "repo", repo)
		metrics.CollectorLocksSkipped.WithLabelValues(repo).Inc()
		return false
	}

	s.log.Debug(i18n.Get(i18n.MsgGhCollectorLockAcquired), "repo", repo, "pod", s.podID)
	metrics.CollectorLocksAcquired.WithLabelValues(repo).Inc()
	return true
}

// collectRepo fetches metadata, workflow runs, and job steps for a single repo.
func (s *GitHubCollectorService) collectRepo(ctx context.Context, owner, repo string) error {
	fullName := owner + "/" + repo
	s.log.Debug(i18n.Get(i18n.MsgGhCollectorRepo), "repo", fullName)

	// ── Repository metadata ──────────────────────────────────────────
	start := time.Now()
	r, _, err := s.ghSvc.Client().Repositories.Get(ctx, owner, repo)
	metrics.ObserveGitHubCall("collector_get_repo", time.Since(start), err)
	if err != nil {
		s.log.Error(i18n.Get(i18n.ErrGhCollectorFetchRepo), "repo", fullName, "error", err)
		return i18n.Err(i18n.ErrGhCollectorFetchRepo, err)
	}

	lang := r.GetLanguage()
	if lang == "" {
		lang = "unknown"
	}
	metrics.SetRepoGauges(
		fullName,
		lang,
		r.GetDefaultBranch(),
		r.GetArchived(),
		r.GetStargazersCount(),
		r.GetForksCount(),
		r.GetOpenIssuesCount(),
		r.GetSize(),
	)

	// ── Workflow runs ────────────────────────────────────────────────
	maxRuns := s.cfg.MaxRecentRuns
	if maxRuns <= 0 {
		maxRuns = 5
	}

	start = time.Now()
	runsResult, _, err := s.ghSvc.Client().Actions.ListRepositoryWorkflowRuns(ctx, owner, repo,
		&github.ListWorkflowRunsOptions{
			ListOptions: github.ListOptions{PerPage: maxRuns},
		},
	)
	metrics.ObserveGitHubCall("collector_list_runs", time.Since(start), err)
	if err != nil {
		s.log.Error(i18n.Get(i18n.ErrGhCollectorFetchRuns), "repo", fullName, "error", err)
		return i18n.Err(i18n.ErrGhCollectorFetchRuns, err)
	}

	for _, run := range runsResult.WorkflowRuns {
		s.collectRun(ctx, owner, repo, fullName, run)
	}

	return nil
}

// collectRun records metrics for a single workflow run and its job steps.
func (s *GitHubCollectorService) collectRun(ctx context.Context, owner, repo, fullName string, run *github.WorkflowRun) {
	runID := fmt.Sprintf("%d", run.GetID())
	workflow := run.GetName()
	conclusion := run.GetConclusion()
	if conclusion == "" {
		conclusion = run.GetStatus() // "in_progress", "queued", etc.
	}

	// Run duration
	if run.RunStartedAt != nil && run.UpdatedAt != nil {
		duration := run.GetUpdatedAt().Time.Sub(run.GetRunStartedAt().Time)
		if duration > 0 {
			metrics.RecordWorkflowRun(fullName, workflow, runID, conclusion, duration)
		}
	}

	// ── Persist workflow run to PostgreSQL ────────────────────────
	if s.pool != nil {
		if runData, err := json.Marshal(run); err == nil {
			if _, err := s.pool.Exec(ctx, dbgithub.UpsertWorkflowRun,
				fullName, run.GetID(), workflow, conclusion, runData, run.GetUpdatedAt().Time,
			); err != nil {
				s.log.Warn("Failed to upsert workflow run", "repo", fullName, "run_id", runID, "error", err)
			}
		}
	}

	// ── Job steps ────────────────────────────────────────────────────
	start := time.Now()
	jobs, _, err := s.ghSvc.Client().Actions.ListWorkflowJobs(ctx, owner, repo, run.GetID(),
		&github.ListWorkflowJobsOptions{
			Filter:      "latest",
			ListOptions: github.ListOptions{PerPage: 100},
		},
	)
	metrics.ObserveGitHubCall("collector_list_jobs", time.Since(start), err)
	if err != nil {
		s.log.Error(i18n.Get(i18n.ErrGhCollectorFetchJobs),
			"repo", fullName, "run_id", runID, "error", err)
		return
	}

	for _, job := range jobs.Jobs {
		jobName := job.GetName()

		// ── Job-level metrics (state-timeline) ──────────────
		jobConclusion := job.GetConclusion()
		if jobConclusion == "" {
			jobConclusion = job.GetStatus()
		}
		jobStarted := job.GetStartedAt().Time
		var jobDur time.Duration
		if job.CompletedAt != nil && !job.GetCompletedAt().Time.IsZero() {
			jobDur = job.GetCompletedAt().Time.Sub(jobStarted)
		}
		metrics.RecordWorkflowJob(fullName, workflow, runID, jobName, jobConclusion, jobStarted, jobDur)

		// ── Persist job to PostgreSQL ────────────────────────
		if s.pool != nil {
			if jobData, err := json.Marshal(job); err == nil {
				if _, err := s.pool.Exec(ctx, dbgithub.UpsertWorkflowJob,
					fullName, run.GetID(), job.GetID(), jobData,
				); err != nil {
					s.log.Warn("Failed to upsert workflow job", "repo", fullName, "job", jobName, "error", err)
				}
			}
		}

		// ── Step-level metrics ───────────────────────────────
		for _, step := range job.Steps {
			stepName := step.GetName()
			stepConclusion := step.GetConclusion()
			if stepConclusion == "" {
				stepConclusion = step.GetStatus()
			}

			startedAt := step.GetStartedAt().Time
			completedAt := step.GetCompletedAt().Time

			// Skip steps that haven't started yet
			if startedAt.IsZero() {
				continue
			}

			var dur time.Duration
			if !completedAt.IsZero() {
				dur = completedAt.Sub(startedAt)
			}

			metrics.RecordWorkflowStep(
				fullName, workflow, runID,
				jobName, stepName, stepConclusion,
				startedAt, dur,
			)
		}

		// ── Annotations for failed jobs ──────────────────────────
		if s.cfg.FetchAnnotations && job.GetConclusion() == "failure" {
			s.collectAnnotations(ctx, owner, repo, fullName, workflow, runID, job)
		}
	}
}

// collectAnnotations fetches check-run annotations for a failed job and records them as metrics.
func (s *GitHubCollectorService) collectAnnotations(ctx context.Context, owner, repo, fullName, workflow, runID string, job *github.WorkflowJob) {
	jobName := job.GetName()
	jobID := job.GetID()

	start := time.Now()
	annotations, _, err := s.ghSvc.Client().Checks.ListCheckRunAnnotations(ctx, owner, repo, jobID,
		&github.ListOptions{PerPage: 100},
	)
	metrics.ObserveGitHubCall("collector_list_annotations", time.Since(start), err)
	if err != nil {
		s.log.Error(i18n.Get(i18n.ErrGhCollectorFetchAnnotations),
			"repo", fullName, "job", jobName, "error", err)
		return
	}

	for _, a := range annotations {
		level := a.GetAnnotationLevel()
		title := a.GetTitle()
		path := a.GetPath()
		metrics.RecordWorkflowAnnotation(fullName, workflow, runID, jobName, level, title, path)
	}

	if len(annotations) > 0 {
		s.log.Debug(i18n.Get(i18n.MsgGhCollectorAnnotations),
			"repo", fullName, "job", jobName, "count", len(annotations))
	}
}

// parseRepo splits "owner/repo" into its parts.
func parseRepo(fullName string) (string, string, error) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("%s: %q", i18n.Get(i18n.ErrGhCollectorParseRepo), fullName)
	}
	return parts[0], parts[1], nil
}
