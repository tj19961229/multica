package agent

import "strings"

// ContextWindowFor returns the max input context size (in tokens) for a model.
// Unknown models fall back to 200_000 — the conservative Anthropic baseline.
// Sources: Anthropic public docs as of 2026-05.
//
// Note: the file is named `model_context_window.go` (singular) on purpose —
// Go's build-tag convention treats `_windows.go` / `_windows_test.go` as
// GOOS=windows-only, so the plural form would silently exclude this file
// from non-Windows builds.
func ContextWindowFor(model string) int64 {
	base := model
	if i := strings.LastIndex(base, "-202"); i >= 0 {
		base = base[:i]
	}
	if w, ok := contextWindows[base]; ok {
		return w
	}
	if w, ok := contextWindows[model]; ok {
		return w
	}
	return 200_000
}

var contextWindows = map[string]int64{
	// 1M context window — GA since 2026-03-13, standard pricing.
	"claude-opus-4-7": 1_000_000,
	// Explicit bracketed alias: ContextWindowFor's date-suffix strip does not
	// touch a trailing "[1m]", so without this entry a "[1m]" model would miss
	// the map and wrongly fall back to 200_000.
	"claude-opus-4-7[1m]": 1_000_000,
	"claude-opus-4-6":     1_000_000,
	"claude-sonnet-4-6":   1_000_000,
	// 200k context window.
	"claude-sonnet-4-5": 200_000,
	"claude-haiku-4-5":  200_000,
	"claude-3-7-sonnet": 200_000,
	"claude-3-5-sonnet": 200_000,
	"claude-3-5-haiku":  200_000,
}
