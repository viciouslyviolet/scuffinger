package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GitHub repository collector metrics.
var (
	// ── Repository metadata gauges ────────────────────────────────────

	// RepoStars reports the star count per repository.
	RepoStars = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "repo_stars",
			Help:      "Number of stars for a monitored repository.",
		},
		[]string{"repo"},
	)

	// RepoForks reports the fork count per repository.
	RepoForks = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "repo_forks",
			Help:      "Number of forks for a monitored repository.",
		},
		[]string{"repo"},
	)

	// RepoOpenIssues reports the open-issue count per repository.
	RepoOpenIssues = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "repo_open_issues",
			Help:      "Number of open issues for a monitored repository.",
		},
		[]string{"repo"},
	)

	// RepoSize reports the repository size in KB.
	RepoSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "repo_size_kb",
			Help:      "Repository size in kilobytes.",
		},
		[]string{"repo"},
	)

	// RepoInfo is a constant-1 info gauge carrying repo metadata as labels.
	RepoInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "repo_info",
			Help:      "Repository metadata (always 1). Labels carry language, default branch, archived status.",
		},
		[]string{"repo", "language", "default_branch", "archived"},
	)

	// ── Workflow run metrics ──────────────────────────────────────────

	// WorkflowRunDuration reports the total duration of a workflow run in seconds.
	WorkflowRunDuration = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_run_duration_seconds",
			Help:      "Total duration of a workflow run in seconds.",
		},
		[]string{"repo", "workflow", "run_id", "conclusion"},
	)

	// WorkflowRunStatus reports the latest run conclusion per workflow (1 = present).
	WorkflowRunStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_run_status",
			Help:      "Latest workflow run status (1 = this conclusion is current).",
		},
		[]string{"repo", "workflow", "conclusion"},
	)

	// ── Workflow step metrics (for Gantt chart) ──────────────────────

	// WorkflowStepDuration reports the duration of a single workflow step in seconds.
	WorkflowStepDuration = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_step_duration_seconds",
			Help:      "Duration of a single workflow job step in seconds.",
		},
		[]string{"repo", "workflow", "run_id", "job", "step", "conclusion"},
	)

	// WorkflowStepStartTime reports the Unix timestamp when a step started.
	WorkflowStepStartTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_step_start_time",
			Help:      "Unix timestamp of when a workflow step started.",
		},
		[]string{"repo", "workflow", "run_id", "job", "step", "conclusion"},
	)

	// ── Workflow job metrics (for state-timeline) ────────────────────

	// WorkflowJobStatus encodes the job conclusion as a number:
	// 0 = success, 1 = failure, 2 = in_progress, 3 = queued, 4 = skipped, 5 = cancelled, -1 = unknown.
	WorkflowJobStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_job_status",
			Help:      "Numeric status of a workflow job (0=success,1=failure,2=in_progress,3=queued,4=skipped,5=cancelled).",
		},
		[]string{"repo", "workflow", "run_id", "job"},
	)

	// WorkflowJobStartedAt reports the Unix timestamp when a job started.
	WorkflowJobStartedAt = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_job_started_at",
			Help:      "Unix timestamp of when a workflow job started.",
		},
		[]string{"repo", "workflow", "run_id", "job"},
	)

	// WorkflowJobDuration reports the duration of a workflow job in seconds.
	WorkflowJobDuration = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_job_duration_seconds",
			Help:      "Duration of a workflow job in seconds.",
		},
		[]string{"repo", "workflow", "run_id", "job"},
	)

	// ── Workflow annotation metrics ──────────────────────────────────

	// WorkflowAnnotationTotal counts annotations by level for a failed workflow run.
	WorkflowAnnotationTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_annotation_total",
			Help:      "Number of annotations on a failed workflow run, by level.",
		},
		[]string{"repo", "workflow", "run_id", "job", "annotation_level"},
	)

	// WorkflowAnnotationInfo is a constant-1 info gauge carrying annotation detail as labels.
	WorkflowAnnotationInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "workflow_annotation_info",
			Help:      "Annotation detail from a failed workflow run (always 1).",
		},
		[]string{"repo", "workflow", "run_id", "job", "annotation_level", "title", "path"},
	)

	// ── Collector cycle counter ──────────────────────────────────────

	CollectorCyclesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "collector_cycles_total",
			Help:      "Total number of collection cycles completed.",
		},
	)

	CollectorErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "collector_errors_total",
			Help:      "Total number of errors during collection, by repo.",
		},
		[]string{"repo"},
	)

	// ── Distributed lock metrics ─────────────────────────────────────

	CollectorLocksAcquired = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "collector_locks_acquired_total",
			Help:      "Total number of distributed locks acquired for repo collection.",
		},
		[]string{"repo"},
	)

	CollectorLocksSkipped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "collector_locks_skipped_total",
			Help:      "Total number of repos skipped because another instance holds the lock.",
		},
		[]string{"repo"},
	)
)

// ── Stale-series management ──────────────────────────────────────────────────

// trackedStepKeys remembers which label-sets we emitted so we can delete stale ones.
var (
	trackedMu             sync.Mutex
	trackedStepKeys       []stepKey
	trackedRunKeys        []runKey
	trackedJobKeys        []jobKey
	trackedAnnotationKeys []annotationKey
	trackedAnnotationInfo []annotationInfoKey
)

type stepKey struct {
	Repo, Workflow, RunID, Job, Step, Conclusion string
}

type runKey struct {
	Repo, Workflow, RunID, Conclusion string
}

type jobKey struct {
	Repo, Workflow, RunID, Job string
}

type annotationKey struct {
	Repo, Workflow, RunID, Job, Level string
}

type annotationInfoKey struct {
	Repo, Workflow, RunID, Job, Level, Title, Path string
}

// ResetTrackedWorkflowMetrics deletes all previously emitted step & run gauge series.
// Call this at the start of every collection cycle before re-populating.
func ResetTrackedWorkflowMetrics() {
	trackedMu.Lock()
	defer trackedMu.Unlock()

	for _, k := range trackedStepKeys {
		WorkflowStepDuration.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Step, k.Conclusion)
		WorkflowStepStartTime.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Step, k.Conclusion)
	}
	for _, k := range trackedRunKeys {
		WorkflowRunDuration.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Conclusion)
	}
	for _, k := range trackedJobKeys {
		WorkflowJobStatus.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job)
		WorkflowJobStartedAt.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job)
		WorkflowJobDuration.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job)
	}
	for _, k := range trackedAnnotationKeys {
		WorkflowAnnotationTotal.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Level)
	}
	for _, k := range trackedAnnotationInfo {
		WorkflowAnnotationInfo.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Level, k.Title, k.Path)
	}
	trackedStepKeys = trackedStepKeys[:0]
	trackedRunKeys = trackedRunKeys[:0]
	trackedJobKeys = trackedJobKeys[:0]
	trackedAnnotationKeys = trackedAnnotationKeys[:0]
	trackedAnnotationInfo = trackedAnnotationInfo[:0]
}

// ResetTrackedWorkflowMetricsForRepos deletes previously emitted gauge series
// only for the given set of repos. Use this in multi-instance deployments where
// each pod only collects a subset of repositories.
func ResetTrackedWorkflowMetricsForRepos(repos map[string]struct{}) {
	trackedMu.Lock()
	defer trackedMu.Unlock()

	// Steps
	kept := trackedStepKeys[:0]
	for _, k := range trackedStepKeys {
		if _, ok := repos[k.Repo]; ok {
			WorkflowStepDuration.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Step, k.Conclusion)
			WorkflowStepStartTime.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Step, k.Conclusion)
		} else {
			kept = append(kept, k)
		}
	}
	trackedStepKeys = kept

	// Runs
	keptRuns := trackedRunKeys[:0]
	for _, k := range trackedRunKeys {
		if _, ok := repos[k.Repo]; ok {
			WorkflowRunDuration.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Conclusion)
		} else {
			keptRuns = append(keptRuns, k)
		}
	}
	trackedRunKeys = keptRuns

	// Jobs
	keptJobs := trackedJobKeys[:0]
	for _, k := range trackedJobKeys {
		if _, ok := repos[k.Repo]; ok {
			WorkflowJobStatus.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job)
			WorkflowJobStartedAt.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job)
			WorkflowJobDuration.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job)
		} else {
			keptJobs = append(keptJobs, k)
		}
	}
	trackedJobKeys = keptJobs

	// Annotations
	keptAnn := trackedAnnotationKeys[:0]
	for _, k := range trackedAnnotationKeys {
		if _, ok := repos[k.Repo]; ok {
			WorkflowAnnotationTotal.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Level)
		} else {
			keptAnn = append(keptAnn, k)
		}
	}
	trackedAnnotationKeys = keptAnn

	// Annotation info
	keptInfo := trackedAnnotationInfo[:0]
	for _, k := range trackedAnnotationInfo {
		if _, ok := repos[k.Repo]; ok {
			WorkflowAnnotationInfo.DeleteLabelValues(k.Repo, k.Workflow, k.RunID, k.Job, k.Level, k.Title, k.Path)
		} else {
			keptInfo = append(keptInfo, k)
		}
	}
	trackedAnnotationInfo = keptInfo
}

// ── Helper setters ───────────────────────────────────────────────────────────

// SetRepoGauges updates all repository metadata gauges.
func SetRepoGauges(repo, language, defaultBranch string, archived bool, stars, forks, openIssues, sizeKB int) {
	RepoStars.WithLabelValues(repo).Set(float64(stars))
	RepoForks.WithLabelValues(repo).Set(float64(forks))
	RepoOpenIssues.WithLabelValues(repo).Set(float64(openIssues))
	RepoSize.WithLabelValues(repo).Set(float64(sizeKB))

	archivedStr := fmt.Sprintf("%t", archived)
	RepoInfo.WithLabelValues(repo, language, defaultBranch, archivedStr).Set(1)
}

// RecordWorkflowRun records the total duration and conclusion of a workflow run.
func RecordWorkflowRun(repo, workflow, runID, conclusion string, duration time.Duration) {
	WorkflowRunDuration.WithLabelValues(repo, workflow, runID, conclusion).Set(duration.Seconds())
	WorkflowRunStatus.WithLabelValues(repo, workflow, conclusion).Set(1)

	trackedMu.Lock()
	trackedRunKeys = append(trackedRunKeys, runKey{repo, workflow, runID, conclusion})
	trackedMu.Unlock()
}

// JobConclusionCode maps a GitHub job conclusion/status string to a numeric code
// for the state-timeline panel: 0=success, 1=failure, 2=in_progress, 3=queued, 4=skipped, 5=cancelled, -1=unknown.
func JobConclusionCode(conclusion string) float64 {
	switch conclusion {
	case "success":
		return 0
	case "failure":
		return 1
	case "in_progress":
		return 2
	case "queued":
		return 3
	case "skipped":
		return 4
	case "cancelled":
		return 5
	default:
		return -1
	}
}

// RecordWorkflowJob records the status, start time, and duration of a workflow job.
func RecordWorkflowJob(repo, workflow, runID, job, conclusion string, startedAt time.Time, duration time.Duration) {
	WorkflowJobStatus.WithLabelValues(repo, workflow, runID, job).Set(JobConclusionCode(conclusion))
	if !startedAt.IsZero() {
		WorkflowJobStartedAt.WithLabelValues(repo, workflow, runID, job).Set(float64(startedAt.Unix()))
	}
	WorkflowJobDuration.WithLabelValues(repo, workflow, runID, job).Set(duration.Seconds())

	trackedMu.Lock()
	trackedJobKeys = append(trackedJobKeys, jobKey{repo, workflow, runID, job})
	trackedMu.Unlock()
}

// RecordWorkflowStep records the start time and duration of a single job step.
func RecordWorkflowStep(repo, workflow, runID, job, step, conclusion string, startedAt time.Time, duration time.Duration) {
	WorkflowStepStartTime.WithLabelValues(repo, workflow, runID, job, step, conclusion).Set(float64(startedAt.Unix()))
	WorkflowStepDuration.WithLabelValues(repo, workflow, runID, job, step, conclusion).Set(duration.Seconds())

	trackedMu.Lock()
	trackedStepKeys = append(trackedStepKeys, stepKey{repo, workflow, runID, job, step, conclusion})
	trackedMu.Unlock()
}

// RecordWorkflowAnnotation records a single annotation from a failed workflow job.
// It increments the per-level counter and sets the info gauge for this annotation.
func RecordWorkflowAnnotation(repo, workflow, runID, job, level, title, path string) {
	WorkflowAnnotationTotal.WithLabelValues(repo, workflow, runID, job, level).Inc()
	WorkflowAnnotationInfo.WithLabelValues(repo, workflow, runID, job, level, title, path).Set(1)

	trackedMu.Lock()
	trackedAnnotationKeys = append(trackedAnnotationKeys, annotationKey{repo, workflow, runID, job, level})
	trackedAnnotationInfo = append(trackedAnnotationInfo, annotationInfoKey{repo, workflow, runID, job, level, title, path})
	trackedMu.Unlock()
}
