package renderer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/fountain/internal/parser"
)

// TestDualDialogueCaretNeverReachesThePage is the regression test for a defect
// nothing covered: the caret is markup, but it used to ride along on the cue text
// and print, so a document using the documented "BRICK ^" syntax rendered a cue
// reading "BRICK ^". The parser now keeps the marker on the element, so the
// printed name is the name.
func TestDualDialogueCaretNeverReachesThePage(t *testing.T) {
	lines := baselineLines(t, "\nINT. HOUSE - DAY\n\nBRICK ^\nBeer!\n")

	var cues []string
	for _, l := range lines {
		if strings.Contains(l.text, "^") {
			t.Fatalf("a caret reached the page: %q", l.text)
		}
		cues = append(cues, l.text)
	}
	if len(cues) < 3 {
		t.Fatalf("baselineLines() = %d lines %v, want heading, cue and dialogue", len(cues), cues)
	}
	if cues[1] != "BRICK" {
		t.Errorf("cue = %q, want %q", cues[1], "BRICK")
	}
	// The cue still lands on its column, and the stripped caret is no longer
	// counted in the name's measured width.
	assertWithin(t, "dual cue left column", lines[1].x/72, 3.7, 0.001)
	assertWithin(t, "dual cue right edge", lines[1].x/72+5*charAdvance(fontSize), 4.2, 0.01)
}

// charAdvance is the width in inches of one character at the given point size.
// Courier Prime is monospaced at 0.5996em, so every character is the same and a
// cue's width is just its length times this.
func charAdvance(size float64) float64 {
	return 0.599609375 * size / 72
}

// TestEveryCueColumnIsCleaner walks the fixture that actually uses the syntax, so
// the caret cannot come back through a path the focused test misses.
func TestEveryCueColumnIsCleaner(t *testing.T) {
	doc, err := parser.ParseFile(filepath.Join("..", "..", "testdata", "all-elements.fountain"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	found := false
	for _, el := range doc.Elements {
		c, ok := el.(parser.Character)
		if !ok {
			continue
		}
		if strings.Contains(c.Text, "^") {
			t.Errorf("cue %q still carries a caret; the parser should record it as Dual", c.Text)
		}
		if c.Dual {
			found = true
		}
	}
	if !found {
		t.Error("the fixture has no dual cue, so this test proves nothing about the marker")
	}
}
