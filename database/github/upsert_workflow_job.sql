INSERT INTO github_workflow_jobs (repo, run_id, job_id, data, fetched_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (job_id) DO UPDATE
    SET data       = EXCLUDED.data,
        fetched_at = now();

