SELECT run_id, workflow, conclusion, data, updated_at, fetched_at
FROM github_workflow_runs
WHERE repo = $1
ORDER BY updated_at DESC
LIMIT $2;

