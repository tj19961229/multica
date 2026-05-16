-- name: UpsertTaskContextState :exec
-- Called from ReportTaskContextUpdate handler at every Claude turn boundary.
-- max_* columns use GREATEST so they only grow within a task's lifetime.
-- latest_* columns are overwritten so the UI can show "current" state too.
INSERT INTO task_context_state (
    task_id, model,
    max_prompt_tokens, max_cache_read_tokens, max_cache_write_tokens,
    latest_prompt_tokens, latest_cache_read_tokens, latest_cache_write_tokens,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $3, $4, $5, now())
ON CONFLICT (task_id) DO UPDATE SET
    model = EXCLUDED.model,
    max_prompt_tokens = GREATEST(task_context_state.max_prompt_tokens, EXCLUDED.max_prompt_tokens),
    max_cache_read_tokens = GREATEST(task_context_state.max_cache_read_tokens, EXCLUDED.max_cache_read_tokens),
    max_cache_write_tokens = GREATEST(task_context_state.max_cache_write_tokens, EXCLUDED.max_cache_write_tokens),
    latest_prompt_tokens = EXCLUDED.latest_prompt_tokens,
    latest_cache_read_tokens = EXCLUDED.latest_cache_read_tokens,
    latest_cache_write_tokens = EXCLUDED.latest_cache_write_tokens,
    updated_at = now();

-- name: GetIssueAgentContexts :many
-- For each agent that has ever run a task on this issue, return the max
-- context size that any of its tasks reached, plus the latest task's status
-- and model so the UI can render a "running"/"completed" badge and pick the
-- right context window via agent.ContextWindowFor.
WITH per_task AS (
    SELECT
        atq.agent_id,
        atq.id AS task_id,
        atq.status,
        tcs.model,
        tcs.max_prompt_tokens + tcs.max_cache_read_tokens AS task_max_context,
        tcs.updated_at,
        ROW_NUMBER() OVER (PARTITION BY atq.agent_id ORDER BY tcs.updated_at DESC) AS rn_latest
    FROM task_context_state tcs
    JOIN agent_task_queue atq ON atq.id = tcs.task_id
    WHERE atq.issue_id = $1
)
SELECT
    pt.agent_id::uuid AS agent_id,
    MAX(pt.task_max_context)::bigint AS max_context_tokens_seen,
    (array_agg(pt.task_id ORDER BY pt.rn_latest))[1]::uuid AS latest_task_id,
    (array_agg(pt.status ORDER BY pt.rn_latest))[1]::text AS latest_task_status,
    (array_agg(pt.model ORDER BY pt.rn_latest))[1]::text AS latest_model
FROM per_task pt
GROUP BY pt.agent_id
ORDER BY MAX(pt.task_max_context) DESC;
