INSERT INTO github_repos (repo, data, fetched_at)
VALUES ($1, $2, now())
ON CONFLICT (repo) DO UPDATE
    SET data       = EXCLUDED.data,
        fetched_at = now();

