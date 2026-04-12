SELECT annotation_level, title, message, path, start_line, end_line, fetched_at
FROM github_annotations
WHERE job_id = $1;

