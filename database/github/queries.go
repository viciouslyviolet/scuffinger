// Package github embeds all SQL queries used by the GitHub data cache.
package github

import _ "embed"

// ── DDL ──────────────────────────────────────────────────────────────────────

//go:embed create_tables.sql
var CreateTables string

// ── Repo ─────────────────────────────────────────────────────────────────────

//go:embed upsert_repo.sql
var UpsertRepo string

//go:embed select_repo.sql
var SelectRepo string

// ── Workflow runs ────────────────────────────────────────────────────────────

//go:embed upsert_workflow_run.sql
var UpsertWorkflowRun string

//go:embed select_recent_runs.sql
var SelectRecentRuns string

// ── Workflow jobs ────────────────────────────────────────────────────────────

//go:embed upsert_workflow_job.sql
var UpsertWorkflowJob string

//go:embed select_jobs_for_run.sql
var SelectJobsForRun string

// ── Annotations ──────────────────────────────────────────────────────────────

//go:embed insert_annotation.sql
var InsertAnnotation string

//go:embed select_annotations_for_job.sql
var SelectAnnotationsForJob string

//go:embed delete_annotations_for_job.sql
var DeleteAnnotationsForJob string
