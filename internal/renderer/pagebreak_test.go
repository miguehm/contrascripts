package renderer

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// The tests here cover where the renderer chooses to break a page, which is a
// different question from the geometry in geometry_test.go: that one asks
// whether a line that was placed landed in the right column, this one asks
// whether it landed on the right page.

// A pageLine is one printed line located on the screenplay grid. The index is
// absolute across the body, so page is index/linesPerPage and the remainder is
// the line's position on that page. Working in grid indices rather than in
// positions within a slice is what lets these tests account for the blank lines
// between elements, which print no text and so are absent from the content
// stream entirely.
type pageLine struct {
	index int
	text  string
	y     float64
	// x is the left edge of the line, which is what the column assertions read.
	x float64
}

func (l pageLine) page() int { return l.index / linesPerPage }

// gridIndex converts a baseline back to its line of the grid.
func gridIndex(y float64) int {
	return int(math.Round((ptPageHeight - ptMarginTop - ptBaselineOff - y) / ptLinePitch))
}

// fillerParagraph builds an action paragraph of exactly n visual lines. Each
// source line is 60 characters, exactly the width of the action column, and the
// parser keeps them as soft breaks inside a single element, so the paragraph
// occupies n lines of the grid and not one line more.
func fillerParagraph(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+"\n", n), "\n")
}

// leavingFree returns a source whose first page carries filler and then head,
// arranged so that exactly free line boxes are left at the foot of page one.
// gap is the number of blank lines the format puts between the filler and the
// head's first element, which is two before a scene heading and one elsewhere.
func leavingFree(head string, gap, free int) string {
	used := linesPerPage - gap - free
	if used < 0 {
		used = 0
	}
	return fillerParagraph(used) + "\n\n" + head + "\n"
}

// bodyLines renders src and returns its lines of script located on the grid.
// The margin bands are skipped: the page number and the CONTINUED: head sit above
// the top margin and the (CONTINUED) foot below the last line, none of which is
// script, and counting the foot as a line would let a page look over budget.
//
// Within a page the baseline climbs as the page fills, so a baseline higher than
// the one before it means a new page started; the blank lines that print no
// text are reconstructed from the line index each baseline reports.
//
// A document with a title page counts that page, so page 0 is the title page
// whenever the source has front matter. Use bodyPageFirst for a per-page view
// keyed the way marginMarksByPage keys it.
func bodyLines(t *testing.T, src string) []pageLine {
	t.Helper()
	var out []pageLine
	page, prevY := 0, 0.0
	for _, l := range baselineLines(t, src) {
		if l.y > ptPageHeight-ptMarginTop || l.y < ptMarginTop-ptBaselineOff {
			continue
		}
		if len(out) > 0 && l.y > prevY {
			page++
		}
		out = append(out, pageLine{
			index: page*linesPerPage + gridIndex(l.y),
			text:  l.text,
			y:     l.y,
			x:     l.x,
		})
		prevY = l.y
	}
	if len(out) == 0 {
		t.Fatal("document produced no body lines, want at least one")
	}
	return out
}

// find returns the line whose text is want, failing the test when absent.
func find(t *testing.T, lines []pageLine, want string) pageLine {
	t.Helper()
	for _, l := range lines {
		if l.text == want {
			return l
		}
	}
	t.Fatalf("no line with text %q in %v", want, textsOfLines(lines))
	return pageLine{}
}

func textsOfLines(lines []pageLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.text
	}
	return out
}

// TestSceneHeadingIsNotOrphaned checks a heading that would be the last thing on
// a page moves down with the action it introduces.
func TestSceneHeadingIsNotOrphaned(t *testing.T) {
	// One line free: not enough for two blank lines, the heading, another blank
	// and the action.
	src := leavingFree("EXT. STREET - NIGHT\nA door stands open.\n", 2, 1)

	lines := bodyLines(t, src)
	heading := find(t, lines, "EXT. STREET - NIGHT")
	action := find(t, lines, "A door stands open.")
	if heading.page() != 1 {
		t.Errorf("scene heading is on page %d line %d, want it moved to page 2",
			heading.page()+1, heading.index)
	}
	if action.page() != heading.page() {
		t.Errorf("action is on page %d, want it on the heading's page %d",
			action.page()+1, heading.page()+1)
	}
}

// TestSceneHeadingKeepsOneLineWithIt checks a heading that has room for the
// blank line and the one line of its action stays at the foot of the page.
func TestSceneHeadingKeepsOneLineWithIt(t *testing.T) {
	src := leavingFree("EXT. STREET - NIGHT\nA door stands open.\n", 2, 3)

	lines := bodyLines(t, src)
	heading := find(t, lines, "EXT. STREET - NIGHT")
	action := find(t, lines, "A door stands open.")
	if heading.page() != 0 {
		t.Fatalf("scene heading is on page %d, want it left on page 1", heading.page()+1)
	}
	if action.page() != 0 {
		t.Errorf("action is on page %d, want it left on page 1", action.page()+1)
	}
	if got, want := action.index-heading.index, 2; got != want {
		t.Errorf("action sits %d lines after the heading, want %d (one blank line)", got, want)
	}
}

// TestStandaloneShotMayEndAPage checks the documented exception: a shot that is
// only a setup, with no content of its own, may end a page alone.
func TestStandaloneShotMayEndAPage(t *testing.T) {
	src := leavingFree("INSERT - A STILL PHOTOGRAPH\n", 2, 1)

	if got := find(t, bodyLines(t, src), "INSERT - A STILL PHOTOGRAPH").page(); got != 0 {
		t.Errorf("insert shot is on page %d, want it left at the foot of page 1", got+1)
	}
}

// TestHeadingMayEndAPageBeforeAnotherHeading checks the other exception: a page
// may end on a heading when another heading follows, which is the establishing
// shot introducing the scene it sets up. The establishing shot is written with
// the forced-heading prefix because the lexer only accepts an EST scene heading
// as "EST." or "EST ".
func TestHeadingMayEndAPageBeforeAnotherHeading(t *testing.T) {
	src := leavingFree(".ESTABLISHING SHOT - THE VALLEY\n\nEXT. CABIN - DAY\n", 2, 2)

	lines := bodyLines(t, src)
	est := find(t, lines, "ESTABLISHING SHOT - THE VALLEY")
	if est.page() != 0 {
		t.Errorf("establishing shot is on page %d, want it left at the foot of page 1", est.page()+1)
	}
	if got := find(t, lines, "EXT. CABIN - DAY").page(); got != 1 {
		t.Errorf("the following heading is on page %d, want page 2", got+1)
	}
}

// TestCharacterCueIsNotOrphaned checks a cue that would be stranded at the foot
// of a page moves down with its speech.
func TestCharacterCueIsNotOrphaned(t *testing.T) {
	src := leavingFree("MARA\nIt is not stopping.\nNot tonight.\n", 1, 1)

	lines := bodyLines(t, src)
	cue := find(t, lines, "MARA")
	if cue.page() != 1 {
		t.Errorf("character cue is on page %d, want it moved to page 2", cue.page()+1)
	}
	if got := find(t, lines, "It is not stopping.").page(); got != 1 {
		t.Errorf("first speech line is on page %d, want it on the cue's page 2", got+1)
	}
}

// TestCharacterCueKeepsTwoLinesWithIt checks a cue with room for two lines of
// speech stays where it is.
func TestCharacterCueKeepsTwoLinesWithIt(t *testing.T) {
	src := leavingFree("MARA\nIt is not stopping.\nNot tonight.\n", 1, 3)

	lines := bodyLines(t, src)
	cue := find(t, lines, "MARA")
	if cue.page() != 0 {
		t.Fatalf("character cue is on page %d, want it left on page 1", cue.page()+1)
	}
	if got := find(t, lines, "Not tonight."); got.page() != 0 {
		t.Errorf("last speech line is on page %d, want it left on page 1", got.page()+1)
	}
}

// TestParentheticalNeverEndsAPage checks a parenthetical is never the last
// thing on a page, since dialogue must follow it.
func TestParentheticalNeverEndsAPage(t *testing.T) {
	// Two lines free: the cue and the parenthetical would fit, nothing after.
	src := leavingFree("MARA\n(whispering)\nIt is not stopping.\n", 1, 2)

	lines := bodyLines(t, src)
	paren := find(t, lines, "(whispering)")
	dialogue := find(t, lines, "It is not stopping.")
	if paren.index%linesPerPage == linesPerPage-1 {
		t.Errorf("parenthetical ends page %d, want it moved", paren.page()+1)
	}
	if paren.page() != dialogue.page() {
		t.Errorf("parenthetical is on page %d and its dialogue on page %d, want them together",
			paren.page()+1, dialogue.page()+1)
	}
}

// TestTransitionNeverStartsAPage checks a transition is never the first line of
// a page; it has to stay with the scene it closes.
func TestTransitionNeverStartsAPage(t *testing.T) {
	for free := 1; free <= 4; free++ {
		src := leavingFree("MARA\nIt is not stopping.\n\nCUT TO:\n", 1, free)
		transition := find(t, bodyLines(t, src), "CUT TO:")
		if got := transition.index % linesPerPage; got == 0 {
			t.Errorf("with %d lines free, the transition is the first line of page %d",
				free, transition.page()+1)
		}
	}
}

// TestSplitPoint covers the decision that governs where a block may be divided:
// the orphan minimum behind, the widow minimum ahead, and the preference for a
// sentence boundary taken as late as the minimums allow.
func TestSplitPoint(t *testing.T) {
	// plain builds a run of n lines, none of which ends a sentence.
	plain := func(n int) []visualLine {
		lines := make([]visualLine, n)
		for i := range lines {
			lines[i] = visualLine{chunks: []styledChunk{{text: "the corridor continues"}}}
		}
		return lines
	}
	// withSentences marks the given zero-based line indices as sentence ends.
	withSentences := func(n int, at ...int) []visualLine {
		lines := plain(n)
		for _, i := range at {
			lines[i].endsSentence = true
		}
		return lines
	}

	tests := []struct {
		name      string
		lines     []visualLine
		available int
		want      int
	}{
		{name: "the whole block fits", lines: plain(4), available: 10, want: 4},
		{name: "the whole block fits exactly", lines: plain(10), available: 10, want: 10},
		{name: "fills the page and splits at the limit", lines: plain(8), available: 6, want: 6},
		{name: "leaves the widow minimum ahead", lines: plain(8), available: 4, want: 4},
		{name: "leaves more than the minimums", lines: plain(8), available: 5, want: 5},
		{name: "no room to leave anything behind", lines: plain(8), available: 1, want: 0},
		{name: "nothing fits at all", lines: plain(8), available: 0, want: 0},
		{name: "prefers the latest sentence boundary", lines: withSentences(8, 3, 5), available: 6, want: 6},
		{name: "walks back to an earlier boundary", lines: withSentences(8, 1), available: 6, want: 2},
		{name: "ignores a boundary below the orphan minimum", lines: withSentences(8, 0), available: 6, want: 6},
		{name: "ignores a boundary past the split limit", lines: withSentences(8, 3, 6), available: 7, want: 4},
		{name: "a single line block is never split", lines: plain(1), available: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitPoint(tt.lines, tt.available); got != tt.want {
				t.Errorf("splitPoint(%d lines, available %d) = %d, want %d",
					len(tt.lines), tt.available, got, tt.want)
			}
		})
	}
}

// TestLongParagraphSplitsAtASentenceBoundary checks the rule end to end on a
// paragraph too long for one page.
func TestLongParagraphSplitsAtASentenceBoundary(t *testing.T) {
	// Sentences of exactly 20 characters, three to a 60 character line, so every
	// line ends on a sentence and the preferred split point is unambiguous.
	const sentence = "The light lies. "
	var b strings.Builder
	for b.Len() < 60*80 {
		b.WriteString(sentence)
	}

	lines := bodyLines(t, b.String())
	firstPage := []pageLine{}
	for _, l := range lines {
		if l.page() == 0 {
			firstPage = append(firstPage, l)
		}
	}
	if len(firstPage) < 2 {
		t.Fatalf("document produced %d lines on page 1, want a full page", len(firstPage))
	}
	last := firstPage[len(firstPage)-1]
	if !strings.HasSuffix(strings.TrimSpace(last.text), ".") {
		t.Errorf("page 1 ends with %q, want a line that finished a sentence", last.text)
	}
	if last.index != linesPerPage-1 {
		t.Errorf("page 1 ends at line %d, want a full page of %d", last.index, linesPerPage)
	}
}

// TestPagesNeverOverflowTheBottomMargin checks the whole grid, line by line, on
// a document long enough to run over several pages.
func TestPagesNeverOverflowTheBottomMargin(t *testing.T) {
	var b strings.Builder
	b.WriteString("INT. HOUSE - DAY\n\n")
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&b, "A door swings open and closes again without a sound, number %03d.\n", i)
	}

	lines := baselineLines(t, b.String())
	if len(lines) < 2 {
		t.Fatalf("document produced %d lines, want a multi-page document", len(lines))
	}
	// A line box may start no lower than the bottom margin, so its baseline is
	// at most that far up the page. Anything lower would be off the sheet.
	lowest := ptMarginTop - ptBaselineOff
	for i, l := range lines {
		if l.y < lowest-0.01 {
			t.Fatalf("line %d at y=%.2f is below the %.2fpt bottom margin", i, l.y, lowest)
		}
	}
}

// TestForcedPageBreakStillWins checks an explicit break in the source is still
// honoured by the new engine.
func TestForcedPageBreakStillWins(t *testing.T) {
	src := "INT. HOUSE - DAY\n\nA door stands open.\n\n===\n\nEXT. STREET - NIGHT\n"
	if got := find(t, bodyLines(t, src), "EXT. STREET - NIGHT").page(); got != 1 {
		t.Errorf("the heading after === is on page %d, want page 2", got+1)
	}
}

// TestHeadingIsFollowedByItsAction reproduces the case where a scene heading
// lands on the last line of a page with a five line action under it. The action
// has to be divided, leaving the orphan minimum behind, because pushing it whole
// would leave the heading alone at the foot of the page with nothing under it.
func TestHeadingIsFollowedByItsAction(t *testing.T) {
	action := strings.TrimSuffix(strings.Repeat("The office is empty and the light is on. ", 2), " ")
	// Sentences of 30 characters, two to a 60 character line, so the action is
	// exactly five lines and its break point is unambiguous.
	action = "Some workers leave their cu" + "biculos, some others stay concentrating on their work. " +
		"Felipe sits at his desk, staring at his monitor."
	action = strings.TrimSpace(action)

	// Leave exactly the heading, the two blank lines above it, the blank line
	// below it and two action lines: five boxes for a heading and two lines of
	// its action.
	src := leavingFree("EXT. OFICINA - NOCHE\n"+action+"\n", 2, 5)

	lines := bodyLines(t, src)
	heading := find(t, lines, "EXT. OFICINA - NOCHE")
	if heading.page() != 0 {
		t.Fatalf("scene heading is on page %d, want it left on page 1", heading.page()+1)
	}

	// The first two lines of the action have to stay under the heading.
	var onPage int
	for _, l := range lines {
		if l.page() == 0 && strings.Contains(l.text, "cubiculos") || (l.page() == 0 && strings.Contains(l.text, "biculos")) {
			onPage++
		}
	}
	if onPage == 0 {
		t.Errorf("no line of the action stayed on page 1; the heading is stranded. Lines: %v", firstFew(lines))
	}
	// Nothing may sit between the heading and the first line of its action.
	first := -1
	for _, l := range lines {
		if l.page() == 0 && l.index > heading.index {
			first = l.index
			break
		}
	}
	if got := first - heading.index; got != 2 {
		t.Errorf("the first line of the action is %d lines after the heading, want 2 (one blank line)", got)
	}
}

// TestActionUnderAHeadingIsDivided checks the division itself: two lines behind,
// the rest over, and no fragment shorter than the minimum on either side.
func TestActionUnderAHeadingIsDivided(t *testing.T) {
	// Twenty sentences of twenty characters, three to a 60 character line, so
	// the paragraph is seven lines. Four of those fit beside the heading, which
	// is fewer than the whole, so it has to be divided.
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("The light lies. ")
	}
	src := leavingFree("EXT. OFICINA - NOCHE\n"+strings.TrimSpace(b.String())+"\n", 2, 5)

	lines := bodyLines(t, src)
	heading := find(t, lines, "EXT. OFICINA - NOCHE")
	if heading.page() != 0 {
		t.Fatalf("scene heading is on page %d, want it left on page 1", heading.page()+1)
	}

	behind, over := 0, 0
	for _, l := range lines {
		if !strings.Contains(l.text, "The light lies.") {
			continue
		}
		if l.page() == 0 {
			behind++
		} else {
			over++
		}
	}
	if behind < orphanLimit {
		t.Errorf("%d lines of the action left on page 1, want at least %d", behind, orphanLimit)
	}
	if over < widowLimit {
		t.Errorf("%d lines carried over to the next page, want at least %d", over, widowLimit)
	}
	if behind+over == 0 {
		t.Error("the action was not divided at all, so the test proves nothing")
	}
}

// bodyPageFirst returns the first line of script on each page, in the same page
// order pagesByY produces, so it lines up one for one with marginMarksByPage.
// The title page is included, because so is its bucket there: comparing a head
// against the page it actually sits on depends on both helpers agreeing on where
// the pages are, and the title page is the one page with no number to split on.
func bodyPageFirst(t *testing.T, src string) []string {
	t.Helper()
	out := make([]string, 0, 8)
	for _, page := range pagesByY(t, src) {
		first := ""
		for _, run := range page {
			if isMarginBand(run.y) {
				continue
			}
			first = run.text
			break
		}
		out = append(out, first)
	}
	return out
}

// pagesByY buckets the rendered operators into pages. Within a page the baseline
// climbs as the page fills, so a baseline higher than the one before it means a
// new page began. This is the same rule baselineLines uses, and it is the only
// page split that does not depend on the page number being present: the title
// page carries none, so keying on it would put the title page and the first page
// of script one apart and silently compare the wrong things.
func pagesByY(t *testing.T, src string) [][]textRun {
	t.Helper()
	var pages [][]textRun
	prevY := 0.0
	for _, run := range textRuns(t, src) {
		if n := len(pages); n == 0 || run.y > prevY+0.5 {
			pages = append(pages, nil)
		}
		pages[len(pages)-1] = append(pages[len(pages)-1], run)
		prevY = run.y
	}
	if len(pages) == 0 {
		t.Fatal("document produced no content, want at least one page")
	}
	return pages
}

// isMarginBand reports whether a run sits outside the block of script, either in
// the header band with the page number or in the footer band with the
// (CONTINUED) foot.
func isMarginBand(y float64) bool {
	return y > ptPageHeight-ptMarginTop || y < ptMarginTop-ptBaselineOff
}
