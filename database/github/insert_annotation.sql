INSERT INTO github_annotations (repo, run_id, job_id, annotation_level, title, message, path, start_line, end_line, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now());

