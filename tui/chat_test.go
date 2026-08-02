package tui

import "testing"

func TestRevealChunkSize(t *testing.T) {
	tests := []struct {
		pending int
		want    int
	}{
		{0, revealBaseRunes},
		{1, revealBaseRunes},
		{revealBaseRunes * revealCatchupDenom, revealBaseRunes}, // right at the boundary
		{revealCatchupDenom * 10, 10},                           // large backlog drains faster
	}
	for _, tt := range tests {
		if got := revealChunkSize(tt.pending); got != tt.want {
			t.Errorf("revealChunkSize(%d) = %d, want %d", tt.pending, got, tt.want)
		}
	}
}

func TestWrapPlain(t *testing.T) {
	if got := wrapPlain("hello", 2); got != "hello" {
		t.Errorf("wrapPlain with tiny width should return input unchanged, got %q", got)
	}
	long := "one two three four five six seven eight nine ten"
	wrapped := wrapPlain(long, 20)
	if wrapped == long {
		t.Errorf("expected wrapPlain to wrap long text at width 20, got unwrapped output")
	}
}

func TestFirstTableLineIndex(t *testing.T) {
	if idx := firstTableLineIndex("just a sentence, no pipes here"); idx != -1 {
		t.Errorf("plain prose should not be detected as a table, got idx=%d", idx)
	}

	s := "intro text\n| Type | Description |\n| --- | --- |"
	idx := firstTableLineIndex(s)
	if idx < 0 || s[idx:idx+1] != "|" {
		t.Errorf("expected idx to point at the table row start, got idx=%d in %q", idx, s)
	}

	// A single line consisting only of a table row: idx should be 0.
	if idx := firstTableLineIndex("| Type | Description |"); idx != 0 {
		t.Errorf("expected idx=0 for a string that starts with a table row, got %d", idx)
	}
}

func TestModelAliasCycling(t *testing.T) {
	if !isKnownModelAlias("fast") || isKnownModelAlias("bogus") {
		t.Errorf("isKnownModelAlias gave wrong result for known/unknown aliases")
	}
	seen := map[string]bool{}
	cur := "auto"
	for range modelAliases {
		cur = nextModelAlias(cur)
		seen[cur] = true
	}
	if len(seen) != len(modelAliases) {
		t.Errorf("cycling through nextModelAlias %d times should visit all %d aliases, saw %d", len(modelAliases), len(modelAliases), len(seen))
	}
}
