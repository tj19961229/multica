package daemon

import (
	"testing"
	"time"
)

// TestShouldThrottleContextUpdate verifies the per-task 500ms throttle:
// the first call for a fresh task_id must pass; immediate follow-ups must
// be throttled until the window elapses; and distinct task_ids must not
// share throttle state.
func TestShouldThrottleContextUpdate(t *testing.T) {
	d := &Daemon{}

	if d.shouldThrottleContextUpdate("task-a") {
		t.Fatalf("first call for task-a must not be throttled")
	}
	if !d.shouldThrottleContextUpdate("task-a") {
		t.Fatalf("immediate second call for task-a must be throttled")
	}

	// A different task must not be throttled by task-a's recent call.
	if d.shouldThrottleContextUpdate("task-b") {
		t.Fatalf("first call for task-b must not be throttled (independent of task-a)")
	}

	// After the throttle window elapses, task-a must pass again. Use a
	// short manual rewind of the stored timestamp so the test doesn't have
	// to sleep 500ms in CI.
	d.contextUpdateLast.Store("task-a", time.Now().Add(-2*contextUpdateThrottle))
	if d.shouldThrottleContextUpdate("task-a") {
		t.Fatalf("call for task-a after throttle window must not be throttled")
	}
}

// TestClearContextUpdateThrottle confirms the cleanup path removes per-task
// state so the throttle map does not grow unbounded over a daemon's lifetime.
func TestClearContextUpdateThrottle(t *testing.T) {
	d := &Daemon{}

	if d.shouldThrottleContextUpdate("task-x") {
		t.Fatalf("first call for task-x must not be throttled")
	}
	if !d.shouldThrottleContextUpdate("task-x") {
		t.Fatalf("immediate second call for task-x must be throttled")
	}

	d.clearContextUpdateThrottle("task-x")

	if d.shouldThrottleContextUpdate("task-x") {
		t.Fatalf("call for task-x after clearContextUpdateThrottle must not be throttled")
	}
}
