package execenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCommentReplyInstructionsIncludesTriggerID(t *testing.T) {
	t.Parallel()

	issueID := "11111111-1111-1111-1111-111111111111"
	triggerID := "22222222-2222-2222-2222-222222222222"

	got := BuildCommentReplyInstructions(issueID, triggerID)

	for _, want := range []string{
		"multica issue comment add " + issueID + " --parent " + triggerID,
		"Always use `--content-stdin`",
		"even when the reply is a single line",
		"--content-stdin",
		"<<'COMMENT'",
		"Do NOT write literal `\\n` escapes to simulate line breaks",
		"do NOT reuse --parent values from previous turns",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("reply instructions missing %q\n---\n%s", want, got)
		}
	}

	if strings.Contains(got, "--content \"...\"") {
		t.Fatalf("reply instructions should not offer inline --content form\n---\n%s", got)
	}
}

func TestBuildCommentReplyInstructionsEmptyWhenNoTrigger(t *testing.T) {
	t.Parallel()

	if got := BuildCommentReplyInstructions("issue-id", ""); got != "" {
		t.Fatalf("expected empty string when triggerCommentID is empty, got %q", got)
	}
}

// TestInjectRuntimeConfigCommentTriggerKeepsClaudeMDStable verifies that
// comment-triggered runs do NOT inline the trigger comment UUID into CLAUDE.md.
// The trigger ID is now delivered via the user message (see buildCommentPrompt
// in prompt.go), keeping CLAUDE.md byte-identical across trigger types so the
// Anthropic prompt-prefix cache survives --resume.
//
// Historically this test asserted the OPPOSITE — that CLAUDE.md inlined both
// the issue ID and the trigger UUID via BuildCommentReplyInstructions. That
// inlining caused the prefix cache to collapse from ~62k to ~18k whenever a
// run was triggered by a new comment. See the cache-stability tests in
// execenv_test.go (TestInjectRuntimeConfigByteIdenticalAcrossTriggers,
// TestInjectRuntimeConfigByteIdenticalAcrossTriggerCommentIDs).
func TestInjectRuntimeConfigCommentTriggerKeepsClaudeMDStable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	issueID := "11111111-1111-1111-1111-111111111111"
	triggerID := "22222222-2222-2222-2222-222222222222"

	ctx := TaskContextForEnv{
		IssueID:          issueID,
		TriggerCommentID: triggerID,
	}
	if _, err := InjectRuntimeConfig(dir, "claude", ctx); err != nil {
		t.Fatalf("InjectRuntimeConfig failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}

	s := string(content)

	// CLAUDE.md MUST NOT contain the trigger comment UUID — that field varies
	// per comment and would break cache prefix on every comment-triggered run.
	if strings.Contains(s, triggerID) {
		t.Errorf("CLAUDE.md must NOT contain trigger comment UUID %q (it varies per run and breaks prompt prefix cache)", triggerID)
	}

	// CLAUDE.md MUST NOT contain a fully-rendered `--parent <UUID>` invocation
	// for the trigger comment. The reply template is generic ("--parent <id>")
	// so the agent looks up the actual ID from the user message.
	if strings.Contains(s, "--parent "+triggerID) {
		t.Errorf("CLAUDE.md must NOT contain rendered --parent %s (rendering pins a per-run UUID into the cache prefix)", triggerID)
	}

	// CLAUDE.md SHOULD still carry generic reply guidance — specifically the
	// instruction to use --parent with the trigger ID delivered via the user
	// message. This is the cache-stable form of the legacy guidance.
	for _, want := range []string{
		"Use the trigger comment ID from the user message as `--parent`",
		"multica issue comment add",
		issueID,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("CLAUDE.md missing generic reply guidance %q", want)
		}
	}
}
