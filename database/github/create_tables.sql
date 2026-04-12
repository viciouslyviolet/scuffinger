-- GitHub data cache tables.
-- Executed at startup with CREATE TABLE IF NOT EXISTS so they are idempotent.

CREATE TABLE IF NOT EXISTS github_repos (
    repo        TEXT        PRIMARY KEY,
    data        JSONB       NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS github_workflow_runs (
    repo        TEXT        NOT NULL,
    run_id      BIGINT      PRIMARY KEY,
    workflow    TEXT        NOT NULL,
    conclusion  TEXT        NOT NULL DEFAULT '',
    data        JSONB       NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_repo ON github_workflow_runs (repo);
CREATE INDEX IF NOT EXISTS idx_workflow_runs_updated ON github_workflow_runs (repo, updated_at DESC);

CREATE TABLE IF NOT EXISTS github_workflow_jobs (
    repo        TEXT        NOT NULL,
    run_id      BIGINT      NOT NULL,
    job_id      BIGINT      PRIMARY KEY,
    data        JSONB       NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_workflow_jobs_run ON github_workflow_jobs (run_id);

CREATE TABLE IF NOT EXISTS github_annotations (
    id               BIGSERIAL   PRIMARY KEY,
    repo             TEXT        NOT NULL,
    run_id           BIGINT      NOT NULL,
    job_id           BIGINT      NOT NULL,
    annotation_level TEXT        NOT NULL,
    title            TEXT        NOT NULL DEFAULT '',
    message          TEXT        NOT NULL DEFAULT '',
    path             TEXT        NOT NULL DEFAULT '',
    start_line       INT         NOT NULL DEFAULT 0,
    end_line         INT         NOT NULL DEFAULT 0,
    fetched_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_annotations_job ON github_annotations (job_id);
CREATE INDEX IF NOT EXISTS idx_annotations_run ON github_annotations (run_id);

