package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// TestReportTaskContextUpdate_PublishesEvent verifies that the
// /api/daemon/tasks/{id}/context-update endpoint is a thin pass-through:
// it loads the task to find agent/issue/workspace IDs, resolves the model's
// max context window, and broadcasts a TaskUsageUpdate event on the bus.
// No DB writes are expected.
func TestReportTaskContextUpdate_PublishesEvent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()

	// Pick the fixture agent and runtime created by TestMain. We then set
	// the agent's model so the handler has a sensible fallback when the
	// request body omits it.
	var agentID, runtimeID string
	if err := testPool.QueryRow(ctx, `
		SELECT a.id, a.runtime_id FROM agent a WHERE a.workspace_id = $1 LIMIT 1
	`, testWorkspaceID).Scan(&agentID, &runtimeID); err != nil {
		t.Fatalf("setup: get agent: %v", err)
	}

	// Restore the agent's model after the test so we don't leak state into
	// other tests that share the fixture.
	var prevModel *string
	if err := testPool.QueryRow(ctx, `SELECT model FROM agent WHERE id = $1`, agentID).Scan(&prevModel); err != nil {
		t.Fatalf("setup: read prior agent model: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `UPDATE agent SET model = $1 WHERE id = $2`, prevModel, agentID)
	})
	if _, err := testPool.Exec(ctx, `UPDATE agent SET model = $1 WHERE id = $2`, "claude-sonnet-4-5", agentID); err != nil {
		t.Fatalf("setup: set agent model: %v", err)
	}

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, status, priority, creator_id, creator_type, number, position)
		VALUES ($1, 'context-update fixture', 'in_progress', 'none', $2, 'member', 82001, 0)
		RETURNING id
	`, testWorkspaceID, testUserID).Scan(&issueID); err != nil {
		t.Fatalf("setup: create issue: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (
			agent_id, runtime_id, issue_id,
			status, priority, started_at
		)
		VALUES ($1, $2, $3, 'running', 0, now())
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("setup: create in_progress task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })

	// Subscribe to the bus BEFORE the call so the synchronous Publish in
	// the handler is captured. The events.Bus is synchronous, so a buffered
	// channel of size 1 + a short wait is sufficient.
	var (
		mu       sync.Mutex
		received []events.Event
	)
	testHandler.Bus.Subscribe(protocol.EventTaskUsageUpdate, func(e events.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	// Snapshot any pre-existing task_usage rows so we can prove the handler
	// does NOT write to the table.
	var preCount int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM task_usage WHERE task_id = $1`, taskID).Scan(&preCount); err != nil {
		t.Fatalf("setup: count task_usage: %v", err)
	}

	w := httptest.NewRecorder()
	req := newDaemonTokenRequest("POST", "/api/daemon/tasks/"+taskID+"/context-update",
		map[string]any{
			"model":              "claude-sonnet-4-6",
			"prompt_tokens":      12500,
			"cache_read_tokens":  80000,
			"cache_write_tokens": 0,
		},
		testWorkspaceID, "legit-daemon")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskId", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	testHandler.ReportTaskContextUpdate(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("ReportTaskContextUpdate: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Bus.Publish is synchronous, but allow a tiny window in case the
	// handler ever moves to async fanout.
	deadline := time.Now().Add(200 * time.Millisecond)
	for {
		mu.Lock()
		got := len(received)
		mu.Unlock()
		if got >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected at least one task:usage_update event, got 0")
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	ev := received[len(received)-1]
	if ev.Type != protocol.EventTaskUsageUpdate {
		t.Fatalf("event type = %q, want %q", ev.Type, protocol.EventTaskUsageUpdate)
	}
	if ev.WorkspaceID != testWorkspaceID {
		t.Fatalf("event workspace_id = %q, want %q", ev.WorkspaceID, testWorkspaceID)
	}
	if ev.TaskID != taskID {
		t.Fatalf("event task_id hint = %q, want %q", ev.TaskID, taskID)
	}
	payload, ok := ev.Payload.(protocol.TaskUsageUpdatePayload)
	if !ok {
		t.Fatalf("event payload type = %T, want protocol.TaskUsageUpdatePayload", ev.Payload)
	}
	if payload.TaskID != taskID {
		t.Fatalf("payload.TaskID = %q, want %q", payload.TaskID, taskID)
	}
	if payload.AgentID != agentID {
		t.Fatalf("payload.AgentID = %q, want %q", payload.AgentID, agentID)
	}
	if payload.IssueID != issueID {
		t.Fatalf("payload.IssueID = %q, want %q", payload.IssueID, issueID)
	}
	if payload.Model != "claude-sonnet-4-6" {
		t.Fatalf("payload.Model = %q, want %q", payload.Model, "claude-sonnet-4-6")
	}
	if payload.PromptTokens != 12500 {
		t.Fatalf("payload.PromptTokens = %d, want %d", payload.PromptTokens, 12500)
	}
	if payload.CacheReadTokens != 80000 {
		t.Fatalf("payload.CacheReadTokens = %d, want %d", payload.CacheReadTokens, 80000)
	}
	if payload.CacheWriteTokens != 0 {
		t.Fatalf("payload.CacheWriteTokens = %d, want %d", payload.CacheWriteTokens, 0)
	}
	if payload.MaxContextTokens <= 0 {
		t.Fatalf("payload.MaxContextTokens = %d, want > 0 (resolved from model)", payload.MaxContextTokens)
	}

	// Confirm no DB writes happened — task_usage rows must be unchanged.
	var postCount int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM task_usage WHERE task_id = $1`, taskID).Scan(&postCount); err != nil {
		t.Fatalf("post-check: count task_usage: %v", err)
	}
	if postCount != preCount {
		t.Fatalf("context-update wrote to task_usage: pre=%d post=%d", preCount, postCount)
	}
}

// TestReportTaskContextUpdate_RejectsNegativeTokens guards the boundary
// validation: a negative token count is a malformed payload and must be
// rejected with 400, not silently broadcast.
func TestReportTaskContextUpdate_RejectsNegativeTokens(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	var agentID, runtimeID string
	if err := testPool.QueryRow(ctx, `
		SELECT a.id, a.runtime_id FROM agent a WHERE a.workspace_id = $1 LIMIT 1
	`, testWorkspaceID).Scan(&agentID, &runtimeID); err != nil {
		t.Fatalf("setup: get agent: %v", err)
	}

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, status, priority, creator_id, creator_type, number, position)
		VALUES ($1, 'context-update neg fixture', 'in_progress', 'none', $2, 'member', 82002, 0)
		RETURNING id
	`, testWorkspaceID, testUserID).Scan(&issueID); err != nil {
		t.Fatalf("setup: create issue: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (
			agent_id, runtime_id, issue_id,
			status, priority, started_at
		)
		VALUES ($1, $2, $3, 'running', 0, now())
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("setup: create in_progress task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })

	cases := []struct {
		name string
		body map[string]any
	}{
		{"prompt_tokens", map[string]any{
			"model":              "claude-sonnet-4-5",
			"prompt_tokens":      -1,
			"cache_read_tokens":  0,
			"cache_write_tokens": 0,
		}},
		{"cache_read_tokens", map[string]any{
			"model":              "claude-sonnet-4-5",
			"prompt_tokens":      0,
			"cache_read_tokens":  -1,
			"cache_write_tokens": 0,
		}},
		{"cache_write_tokens", map[string]any{
			"model":              "claude-sonnet-4-5",
			"prompt_tokens":      0,
			"cache_read_tokens":  0,
			"cache_write_tokens": -1,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := newDaemonTokenRequest("POST", "/api/daemon/tasks/"+taskID+"/context-update",
				tc.body,
				testWorkspaceID, "legit-daemon")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("taskId", taskID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			testHandler.ReportTaskContextUpdate(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s: status = %d, want 400; body=%s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}
