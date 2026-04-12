INSERT INTO github_workflow_runs (repo, run_id, workflow, conclusion, data, updated_at, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (run_id) DO UPDATE
    SET workflow    = EXCLUDED.workflow,
        conclusion  = EXCLUDED.conclusion,
        data        = EXCLUDED.data,
        updated_at  = EXCLUDED.updated_at,
        fetched_at  = now();

