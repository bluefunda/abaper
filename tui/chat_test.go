package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
)

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

func TestLooksLikeTable(t *testing.T) {
	if !looksLikeTable("| Type | Description |") {
		t.Errorf("expected a line starting with | to be detected as a table")
	}
	if !looksLikeTable("intro text\n| Type | Description |\n| --- | --- |") {
		t.Errorf("expected a multi-line block containing a table row to be detected")
	}
	if looksLikeTable("just a sentence, no pipes here") {
		t.Errorf("plain prose should not be detected as a table")
	}
}

func newTestChatModel(t *testing.T, width int) *chatModel {
	t.Helper()
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(width-6))
	if err != nil {
		t.Fatal(err)
	}
	return &chatModel{renderer: r, width: width}
}

// TestRenderStreamingContent_ResumesLiveFormattingAfterTable is a regression
// test for a bug where, once any table started streaming, every later
// paragraph and table in the same response stayed in plain-text mode for
// the rest of the stream — a completed table (or paragraph), once followed
// by a real blank line, must go back to rendering live via glamour.
func TestRenderStreamingContent_ResumesLiveFormattingAfterTable(t *testing.T) {
	m := newTestChatModel(t, 94)

	settledPart := "## Heading\n\nIntro paragraph.\n\n" +
		"| A | B |\n|---|---|\n| 1 | 2 |\n\n" +
		"## Second heading\n\n"
	stillWriting := "More prose after the tab" // no blank line after this yet

	// Mid-write: the second heading's section hasn't reached a blank-line
	// boundary yet, so it's the "current" block and renders live as prose
	// (not held back — only an in-progress *table* is held back).
	out := m.renderStreamingContent(settledPart + stillWriting)
	if strings.Contains(out, "## Second heading") {
		t.Errorf("heading markdown should have been consumed by the live glamour render, got literal '##' in: %q", out)
	}
	if strings.Contains(out, "|---|") {
		t.Errorf("the completed table (followed by a blank line) should have rendered live, not shown as raw pipes: %q", out)
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
