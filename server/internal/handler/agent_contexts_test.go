package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestReportTaskContextUpdate_PersistsToDB verifies the v3 addition: after the
// existing WS publish, the handler also writes max/latest into task_context_state.
// max columns must take GREATEST across calls; latest columns must overwrite.
func TestReportTaskContextUpdate_PersistsToDB(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()

	var runtimeID, agentID string
	if err := testPool.QueryRow(ctx, `SELECT id FROM agent_runtime WHERE workspace_id = $1 LIMIT 1`, testWorkspaceID).Scan(&runtimeID); err != nil {
		t.Fatalf("fetch runtime: %v", err)
	}
	if err := testPool.QueryRow(ctx, `SELECT id FROM agent WHERE workspace_id = $1 LIMIT 1`, testWorkspaceID).Scan(&agentID); err != nil {
		t.Fatalf("fetch agent: %v", err)
	}

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, status, priority, creator_id, creator_type, number, position)
		VALUES ($1, 'persist test', 'in_progress', 'none', $2, 'member', 82101, 0)
		RETURNING id
	`, testWorkspaceID, testUserID).Scan(&issueID); err != nil {
		t.Fatalf("create issue: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, started_at)
		VALUES ($1, $2, $3, 'running', 0, now())
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("create task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })

	callContextUpdate := func(prompt, cacheRead, cacheWrite int64) {
		body := map[string]any{
			"model":              "claude-sonnet-4-6",
			"prompt_tokens":      prompt,
			"cache_read_tokens":  cacheRead,
			"cache_write_tokens": cacheWrite,
		}
		w := httptest.NewRecorder()
		req := newDaemonTokenRequest("POST", "/api/daemon/tasks/"+taskID+"/context-update",
			body, testWorkspaceID, "legit-daemon")
		req = withURLParam(req, "taskId", taskID)
		testHandler.ReportTaskContextUpdate(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
		}
	}

	// First call: max = latest = (1500, 800, 0)
	callContextUpdate(1500, 800, 0)

	var maxP, maxR, maxW, lastP, lastR, lastW int64
	var model string
	if err := testPool.QueryRow(ctx, `
		SELECT model, max_prompt_tokens, max_cache_read_tokens, max_cache_write_tokens,
		       latest_prompt_tokens, latest_cache_read_tokens, latest_cache_write_tokens
		FROM task_context_state WHERE task_id = $1
	`, taskID).Scan(&model, &maxP, &maxR, &maxW, &lastP, &lastR, &lastW); err != nil {
		t.Fatalf("after 1st call, fetch row: %v", err)
	}
	if model != "claude-sonnet-4-6" || maxP != 1500 || maxR != 800 || maxW != 0 || lastP != 1500 || lastR != 800 || lastW != 0 {
		t.Fatalf("after 1st call: row mismatch model=%s max=(%d,%d,%d) latest=(%d,%d,%d)", model, maxP, maxR, maxW, lastP, lastR, lastW)
	}

	// Second call: smaller prompt — max stays, latest drops.
	callContextUpdate(1000, 600, 0)
	if err := testPool.QueryRow(ctx, `
		SELECT max_prompt_tokens, max_cache_read_tokens, latest_prompt_tokens, latest_cache_read_tokens
		FROM task_context_state WHERE task_id = $1
	`, taskID).Scan(&maxP, &maxR, &lastP, &lastR); err != nil {
		t.Fatalf("after 2nd call, fetch row: %v", err)
	}
	if maxP != 1500 || maxR != 800 {
		t.Fatalf("after 2nd call (smaller prompt): max should stay (1500,800) got (%d,%d)", maxP, maxR)
	}
	if lastP != 1000 || lastR != 600 {
		t.Fatalf("after 2nd call: latest should overwrite to (1000,600) got (%d,%d)", lastP, lastR)
	}

	// Third call: larger prompt — max should grow.
	callContextUpdate(2000, 700, 0)
	if err := testPool.QueryRow(ctx, `SELECT max_prompt_tokens FROM task_context_state WHERE task_id = $1`, taskID).Scan(&maxP); err != nil {
		t.Fatalf("after 3rd call: %v", err)
	}
	if maxP != 2000 {
		t.Fatalf("after 3rd call (larger prompt): max should grow to 2000 got %d", maxP)
	}
}

// TestGetIssueAgentContexts_AggregatesPerAgent verifies the new GET endpoint
// returns one row per agent with max-of-tasks aggregation, plus the latest
// task's status/model so the UI can pick the right context window.
func TestGetIssueAgentContexts_AggregatesPerAgent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()

	var runtimeID, agentA, agentB string
	if err := testPool.QueryRow(ctx, `SELECT id FROM agent_runtime WHERE workspace_id = $1 LIMIT 1`, testWorkspaceID).Scan(&runtimeID); err != nil {
		t.Fatalf("fetch runtime: %v", err)
	}
	if err := testPool.QueryRow(ctx, `SELECT id FROM agent WHERE workspace_id = $1 LIMIT 1`, testWorkspaceID).Scan(&agentA); err != nil {
		t.Fatalf("fetch agent A: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (workspace_id, name, description, runtime_mode, runtime_config, runtime_id, visibility, max_concurrent_tasks, owner_id, instructions, custom_env, custom_args)
		VALUES ($1, 'agent-contexts-test-b', '', 'local', '{}'::jsonb, $2, 'private', 1, $3, '', '{}'::jsonb, '[]'::jsonb)
		RETURNING id
	`, testWorkspaceID, runtimeID, testUserID).Scan(&agentB); err != nil {
		t.Fatalf("create agent B: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentB) })

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, status, priority, creator_id, creator_type, number, position)
		VALUES ($1, 'agent-contexts agg test', 'in_progress', 'none', $2, 'member', 82102, 0)
		RETURNING id
	`, testWorkspaceID, testUserID).Scan(&issueID); err != nil {
		t.Fatalf("create issue: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

	insertTaskWithMax := func(agentID, status string, maxPrompt int64) string {
		var taskID string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, started_at)
			VALUES ($1, $2, $3, $4, 0, now())
			RETURNING id
		`, agentID, runtimeID, issueID, status).Scan(&taskID); err != nil {
			t.Fatalf("create task: %v", err)
		}
		if _, err := testPool.Exec(ctx, `
			INSERT INTO task_context_state (task_id, model, max_prompt_tokens, max_cache_read_tokens, latest_prompt_tokens)
			VALUES ($1, 'claude-sonnet-4-6', $2, 0, $2)
		`, taskID, maxPrompt); err != nil {
			t.Fatalf("insert state: %v", err)
		}
		t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })
		return taskID
	}

	insertTaskWithMax(agentA, "completed", 1500)
	insertTaskWithMax(agentA, "running", 2500) // newer + bigger
	insertTaskWithMax(agentB, "failed", 800)

	w := httptest.NewRecorder()
	req := newRequest("GET", "/api/issues/"+issueID+"/agent-contexts", nil)
	req = withURLParam(req, "id", issueID)
	testHandler.GetIssueAgentContexts(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}

	var rows []struct {
		AgentID              string `json:"agent_id"`
		MaxContextTokensSeen int64  `json:"max_context_tokens_seen"`
		LatestTaskID         string `json:"latest_task_id"`
		LatestTaskStatus     string `json:"latest_task_status"`
		LatestModel          string `json:"latest_model"`
		MaxContextTokens     int64  `json:"max_context_tokens"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 agent rows, got %d (%s)", len(rows), w.Body.String())
	}
	byAgent := map[string]int64{}
	statuses := map[string]string{}
	for _, r := range rows {
		byAgent[r.AgentID] = r.MaxContextTokensSeen
		statuses[r.AgentID] = r.LatestTaskStatus
		if r.MaxContextTokens != 200_000 {
			t.Errorf("expected max_context_tokens=200000 for sonnet model, got %d", r.MaxContextTokens)
		}
	}
	if byAgent[agentA] != 2500 {
		t.Fatalf("agent A max should be 2500 (max of 1500 and 2500), got %d", byAgent[agentA])
	}
	if byAgent[agentB] != 800 {
		t.Fatalf("agent B max should be 800, got %d", byAgent[agentB])
	}
	if statuses[agentA] != "running" {
		t.Errorf("agent A latest status should be 'running' (newest task), got %q", statuses[agentA])
	}
	if statuses[agentB] != "failed" {
		t.Errorf("agent B latest status should be 'failed', got %q", statuses[agentB])
	}
}
