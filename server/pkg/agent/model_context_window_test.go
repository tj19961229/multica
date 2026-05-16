package agent

import "testing"

func TestContextWindowFor(t *testing.T) {
	cases := []struct {
		name  string
		model string
		want  int64
	}{
		{"claude opus 4.7", "claude-opus-4-7", 1_000_000},
		{"claude opus 4.7 1m", "claude-opus-4-7[1m]", 1_000_000},
		{"claude sonnet 4.6", "claude-sonnet-4-6", 1_000_000},
		{"claude haiku 4.5 dated", "claude-haiku-4-5-20251001", 200_000},
		{"claude sonnet 4.5", "claude-sonnet-4-5", 200_000},
		{"unknown model fallback", "totally-unknown-model", 200_000},
		{"empty string fallback", "", 200_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ContextWindowFor(tc.model)
			if got != tc.want {
				t.Fatalf("%s = %d, want %d", tc.model, got, tc.want)
			}
		})
	}
}
