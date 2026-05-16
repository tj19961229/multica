-- Per-task snapshot of conversation context size. Distinct from task_usage
-- which holds cumulative billing-style token totals. Each task gets one row;
-- daemon's per-turn ReportTaskContextUpdate calls UPSERT this row, taking
-- GREATEST for the max columns and overwriting the latest columns.
--
-- "Context size" semantically = the prompt size of a single Claude turn,
-- including cache_read tokens (which still consume the context window).
-- max_prompt_tokens / max_cache_read_tokens together represent the largest
-- context window occupancy this task ever reached.
CREATE TABLE task_context_state (
    task_id UUID PRIMARY KEY REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    model TEXT NOT NULL DEFAULT '',
    max_prompt_tokens BIGINT NOT NULL DEFAULT 0,
    max_cache_read_tokens BIGINT NOT NULL DEFAULT 0,
    max_cache_write_tokens BIGINT NOT NULL DEFAULT 0,
    latest_prompt_tokens BIGINT NOT NULL DEFAULT 0,
    latest_cache_read_tokens BIGINT NOT NULL DEFAULT 0,
    latest_cache_write_tokens BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_context_state_updated_at ON task_context_state(updated_at);
