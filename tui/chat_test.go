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

func TestSplitTrailingParagraph(t *testing.T) {
	settled, trailing := splitTrailingParagraph("para one.\n\npara two still writing")
	if settled != "para one.\n\n" || trailing != "para two still writing" {
		t.Errorf("got settled=%q trailing=%q", settled, trailing)
	}

	settled, trailing = splitTrailingParagraph("no blank line yet")
	if settled != "" || trailing != "no blank line yet" {
		t.Errorf("with no blank line, everything should be trailing; got settled=%q trailing=%q", settled, trailing)
	}
}

func TestLooksLikeInProgressTable(t *testing.T) {
	if !looksLikeInProgressTable("| Type | Description |") {
		t.Errorf("expected a line starting with | to be detected as a table")
	}
	if !looksLikeInProgressTable("intro text\n| Type | Description |\n| --- | --- |") {
		t.Errorf("expected a multi-line block containing a table row to be detected")
	}
	if looksLikeInProgressTable("just a sentence, no pipes here") {
		t.Errorf("plain prose should not be detected as a table")
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
