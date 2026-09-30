package renderer

import (
	"github.com/user/fountain/internal/parser"
	"strings"
	"testing"
)

// Continuation notation belongs to a production script only. These tests check
// both halves of that: that `production: true` turns the marks on, and that
// nothing appears without it.

// frontMatter wraps body in the YAML front matter the format reads, enabling the
// production notation when asked.
func frontMatter(production bool, body string) string {
	flag := "production: false\n"
	if production {
		flag = "production: true\n"
	}
	return "---\ntitle: Test\nauthor: Nobody\n" + flag + "---\n\n" + body
}

// speechOf returns a speech of 20 character sentences, three to a 60 character
// line, long enough that it cannot fit on one page.
func speechOf(lines int) string {
	var b strings.Builder
	for i := 0; i < lines*3; i++ {
		b.WriteString("The light lies. ")
	}
	return strings.TrimSpace(b.String())
}

// productionSource builds a script whose speech is long enough to be divided,
// with the requested title page.
func productionSource(production bool) string {
	body := "INT. HOUSE - DAY\n\nA door stands open.\n\nMARA\n" + speechOf(40) + "\n"
	return frontMatter(production, body)
}

func renderedLines(t *testing.T, src string) []pageLine {
	t.Helper()
	return bodyLines(t, src)
}

func containsLine(lines []pageLine, want string) bool {
	for _, l := range lines {
		if l.text == want {
			return true
		}
	}
	return false
}

// TestProductionScriptMarksSplitSpeech checks that a divided speech carries
// (MORE) at the foot of the first page and the cue with (CONT'D) above the
// remainder.
func TestProductionScriptMarksSplitSpeech(t *testing.T) {
	lines := renderedLines(t, productionSource(true))

	const speech = "The light lies."
	more := find(t, lines, moreText)
	contd := find(t, lines, "MARA "+contdText)

	if more.page() == contd.page() {
		t.Errorf("(MORE) is on page %d and the continued cue too, want them on consecutive pages",
			more.page()+1)
	}
	// (MORE) goes directly under the dialogue it belongs to, with no blank
	// line between them, and the repeated cue directly above the remainder.
	assertFollows(t, lines, more.index-1, speech, "the line before (MORE)")
	assertFollows(t, lines, contd.index+1, speech, "the line after the continued cue")
	if got := contd.index % linesPerPage; got != 0 {
		t.Errorf("the continued cue is at line %d of page %d, want it at the top",
			got, contd.page()+1)
	}
	// The speech has to be divided for any of this to mean anything.
	if _, last := firstAndLast(t, lines, speech); last.page() <= more.page() {
		t.Error("the speech was not divided across pages, so the test proves nothing")
	}
}

// TestSpecScriptOmitsContinuation checks a script without `production: true`
// carries none of the notation.
func TestSpecScriptOmitsContinuation(t *testing.T) {
	lines := renderedLines(t, productionSource(false))

	for _, unwanted := range []string{moreText, "MARA " + contdText, continuedTx, continuedHd} {
		if containsLine(lines, unwanted) {
			t.Errorf("line %q appears in a spec script, want it only in a production script", unwanted)
		}
	}
	// The speech still has to be divided across pages, or the test proves
	// nothing about the notation being suppressed.
	if pages := speechPages(lines); len(pages) < 2 {
		t.Errorf("the speech occupies %d page(s), want it divided so the test is meaningful", len(pages))
	}
}

// assertFollows checks the line at index carries want.
func assertFollows(t *testing.T, lines []pageLine, index int, want, what string) {
	t.Helper()
	for _, l := range lines {
		if l.index != index {
			continue
		}
		if !strings.Contains(l.text, want) {
			t.Errorf("%s = %q, want it to contain %q", what, l.text, want)
		}
		return
	}
	t.Errorf("%s: no line at grid index %d, want one containing %q", what, index, want)
}

// firstAndLast returns the first and last lines containing want.
func firstAndLast(t *testing.T, lines []pageLine, want string) (pageLine, pageLine) {
	t.Helper()
	var first, last pageLine
	seen := false
	for _, l := range lines {
		if !strings.Contains(l.text, want) {
			continue
		}
		if !seen {
			first, seen = l, true
		}
		last = l
	}
	if !seen {
		t.Fatalf("no line containing %q", want)
	}
	return first, last
}

// speechPages returns the distinct pages carrying dialogue text, which for this
// document is the repeated sentence.
func speechPages(lines []pageLine) map[int]bool {
	pages := map[int]bool{}
	for _, l := range lines {
		if strings.Contains(l.text, "The light lies.") {
			pages[l.page()] = true
		}
	}
	return pages
}

// TestProductionScriptMarksSplitScene checks a scene that runs past the foot of
// a page carries (CONTINUED) there and CONTINUED: n in the head band of the next.
func TestProductionScriptMarksSplitScene(t *testing.T) {
	var b strings.Builder
	b.WriteString("INT. HOUSE - DAY\n\n")
	for b.Len() < 60*80 {
		b.WriteString("A door swings open. ")
	}
	b.WriteString("\n\nEXT. STREET - NIGHT\n\nA street stands empty.\n")

	m := collectMarginMarks(t, frontMatter(true, b.String()))
	if len(m.feet) == 0 {
		t.Errorf("no %s on the page the scene runs past", continuedTx)
	}
	if len(m.heads) == 0 {
		t.Errorf("no %s in the head band of a continued page", continuedHd)
	}
	// The two marks come in pairs: every page that carries a foot is followed by
	// one that carries a head.
	if len(m.feet) != len(m.heads) {
		t.Errorf("%d feet but %d heads, want one head per continued page", len(m.feet), len(m.heads))
	}
}

func hasPrefixLine(lines []pageLine, prefix string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l.text, prefix) {
			return true
		}
	}
	return false
}

func firstFew(lines []pageLine) []string {
	if len(lines) > 6 {
		lines = lines[:6]
	}
	return textsOfLines(lines)
}

// TestContinuedCueKeepsAnExistingExtension checks a name that already carries an
// extension gets the (CONT'D) after it rather than losing the first one.
func TestContinuedCueKeepsAnExistingExtension(t *testing.T) {
	tests := []struct {
		name string
		in   parser.Character
		want string
	}{
		{name: "plain name gains the extension", in: parser.Character{Text: "MARA"}, want: "MARA (CONT'D)"},
		{name: "voice over keeps its extension", in: parser.Character{Text: "MARA (V.O.)"}, want: "MARA (V.O.); " + contdText},
		{name: "off screen keeps its extension", in: parser.Character{Text: "MARA (O.S.)"}, want: "MARA (O.S.); " + contdText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := continuedCue(tt.in).Text; got != tt.want {
				t.Errorf("continuedCue(%q) = %q, want %q", tt.in.Text, got, tt.want)
			}
		})
	}
}

// marginMarks returns the text a production script prints in the bands above and
// below the text block, keyed by kind. The bodyLines helper deliberately drops
// everything up there, because that band is where the page number lives and a
// test about pagination has no business counting it as script.
type marginMarks struct {
	heads     []pageLine // CONTINUED: n
	feet      []pageLine // (CONTINUED)
	pageNums  []pageLine
	allInHead bool
}

func collectMarginMarks(t *testing.T, src string) marginMarks {
	t.Helper()
	var m marginMarks
	// Raw runs rather than grouped lines: the CONTINUED: head and the page
	// number are separate cells that share the header band's baseline, and
	// grouping them would read as one string.
	for _, run := range textRuns(t, src) {
		line := pageLine{text: run.text, y: run.y, x: run.x}
		switch {
		case run.y > ptPageHeight-ptMarginTop:
			// The band above the top margin: page numbers and CONTINUED: heads.
			m.pageNums = append(m.pageNums, line)
			if strings.HasPrefix(run.text, continuedHd) {
				m.heads = append(m.heads, line)
			}
		case run.y < ptMarginTop-ptBaselineOff:
			// Below the last line of script: the (CONTINUED) foot.
			if run.text == continuedTx {
				m.feet = append(m.feet, line)
			}
		}
	}
	return m
}

// TestContinuedMarksSitOutsideTheTextBlock is the regression test for the
// overlapping characters: neither mark may share a baseline with body text, so
// both have to live in the bands around the text block rather than inside it.
func TestContinuedMarksSitOutsideTheTextBlock(t *testing.T) {
	var b strings.Builder
	b.WriteString("INT. HOUSE - DAY\n\n")
	for b.Len() < 60*80 {
		b.WriteString("A door swings open. ")
	}
	b.WriteString("\n\nEXT. STREET - NIGHT\n\nA street stands empty.\n")

	lines := baselineLines(t, frontMatter(true, b.String()))
	if len(lines) < 2 {
		t.Fatalf("document produced %d lines, want a multi-page script", len(lines))
	}

	bodyBand := func(y float64) bool {
		return y <= ptPageHeight-ptMarginTop && y >= ptMarginTop-ptBaselineOff
	}
	for _, l := range lines {
		if bodyBand(l.y) {
			continue
		}
		// Nothing printed outside the text block may share a baseline with
		// anything else at all, so a collision would show up as a repeat.
		for _, other := range lines {
			if other == l {
				continue
			}
			if l.text == other.text && l.y == other.y {
				t.Fatalf("mark %q at y=%.2f is printed twice on the same baseline", l.text, l.y)
			}
		}
	}

	m := collectMarginMarks(t, frontMatter(true, b.String()))
	if len(m.heads) == 0 {
		t.Error("no CONTINUED: head found, want one on every page the scene runs past")
	}
	if len(m.feet) == 0 {
		t.Error("no (CONTINUED) foot found, want one on every page the scene runs past")
	}
	// The head must name the scene, not the page: the same scene spans several
	// pages and every one of them repeats the same number.
	if len(m.heads) > 1 {
		if m.heads[0].text != m.heads[len(m.heads)-1].text {
			t.Errorf("heads across a scene read %q then %q, want the same scene number throughout",
				m.heads[0].text, m.heads[len(m.heads)-1].text)
		}
	}
	if want := continuedHd + " 1"; m.heads[0].text != want {
		t.Errorf("first head = %q, want %q", m.heads[0].text, want)
	}
}

// TestContinuedMarksSurviveDialogue is the regression test for a scene that runs
// past the foot of a page after the character has started speaking: the scene is
// still open, so the page must still be marked.
func TestContinuedMarksSurviveDialogue(t *testing.T) {
	var b strings.Builder
	b.WriteString("INT. HOUSE - DAY\n\nA door swings open. ")
	for b.Len() < 60*80 {
		b.WriteString("A door swings open. ")
	}
	b.WriteString("\n\nMARA\n" + speechOf(40) + "\n")

	m := collectMarginMarks(t, frontMatter(true, b.String()))
	if len(m.heads) == 0 {
		t.Error("no CONTINUED: head after a page break that follows dialogue, want one")
	}
	if len(m.feet) == 0 {
		t.Error("no (CONTINUED) foot after a page break that follows dialogue, want one")
	}
}

// TestSceneThatFitsOnePageHasNoMarks checks the marks only appear where a scene
// really does run past the foot of a page.
func TestSceneThatFitsOnePageHasNoMarks(t *testing.T) {
	src := frontMatter(true, "INT. HOUSE - DAY\n\nA door stands open.\n\nMARA\nIt is not stopping.\n")
	m := collectMarginMarks(t, src)
	if len(m.heads) != 0 {
		t.Errorf("heads = %v, want none for a scene that fits one page", textsOfLines(m.heads))
	}
	if len(m.feet) != 0 {
		t.Errorf("feet = %v, want none for a scene that fits one page", textsOfLines(m.feet))
	}
}

// TestNoMarksAfterATransition checks a transition closes the scene, so the next
// page is the start of a new one and carries no continuation notation.
func TestNoMarksAfterATransition(t *testing.T) {
	var b strings.Builder
	b.WriteString("INT. HOUSE - DAY\n\n")
	for b.Len() < 60*80 {
		b.WriteString("A door swings open. ")
	}
	b.WriteString("\n\nCUT TO:\n\nEXT. STREET - NIGHT\n\nA street stands empty.\n")

	m := collectMarginMarks(t, frontMatter(true, b.String()))
	// The new scene is one line long, so it cannot run past a foot. Nothing on
	// that page may claim to continue the previous one.
	if last := m.heads[len(m.heads)-1]; last.text != continuedHd+" 1" {
		t.Errorf("last head = %q, want it still to name scene 1 only while that scene is open", last.text)
	}
	for _, f := range m.feet {
		if f.text != continuedTx {
			t.Errorf("unexpected foot mark %q", f.text)
		}
	}
}

// TestHeadNamesTheSceneNotThePage checks the number after CONTINUED: is the
// scene's ordinal, and is independent of whether scene numbers are drawn in the
// margins.
func TestHeadNamesTheSceneNotThePage(t *testing.T) {
	var b strings.Builder
	b.WriteString("INT. HOUSE - DAY\n\n")
	for b.Len() < 60*80 {
		b.WriteString("A door swings open. ")
	}
	b.WriteString("\n\nEXT. CABIN - DAY\n\nA cabin stands empty.\n\n")
	for b.Len() < 60*140 {
		b.WriteString("A door swings open. ")
	}
	b.WriteString("\n")

	// No scene-numbers: the head must still name the scenes it can see.
	noNums := collectMarginMarks(t, frontMatter(true, b.String()))
	if len(noNums.heads) == 0 {
		t.Fatal("no heads found, want at least one")
	}
	if !hasPrefixLine(noNums.heads, continuedHd+" 1") {
		t.Errorf("heads = %v, want one naming scene 1", textsOfLines(noNums.heads))
	}

	// The same document with scene numbers in the margins must say the same thing,
	// because the two features are independent.
	nums := collectMarginMarks(t, frontMatter(true, strings.Replace(b.String(),
		"production: true", "production: true\nscene-numbers: true", 1)))
	if got, want := textsOfLines(nums.heads), textsOfLines(noNums.heads); !equalStrings(got, want) {
		t.Errorf("heads with scene-numbers = %v, want %v, the notation should not depend on it", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// marginMarksByPage collects the margin marks page by page, using the same page
// split as bodyPageFirst so a head is always compared against the page it is
// printed on.
func marginMarksByPage(t *testing.T, src string) []marginMarks {
	t.Helper()
	pages := pagesByY(t, src)
	out := make([]marginMarks, 0, len(pages))
	for _, page := range pages {
		var m marginMarks
		for _, run := range page {
			line := pageLine{text: run.text, y: run.y, x: run.x}
			switch {
			case run.y > ptPageHeight-ptMarginTop:
				m.pageNums = append(m.pageNums, line)
				if strings.HasPrefix(run.text, continuedHd) {
					m.heads = append(m.heads, line)
				}
			case run.y < ptMarginTop-ptBaselineOff && run.text == continuedTx:
				m.feet = append(m.feet, line)
			}
		}
		out = append(out, m)
	}
	return out
}

// TestHeadingPushedToANewPageCarriesNoHead covers the case the head flag exists
// for: a scene heading that does not fit at the foot of a page is pushed whole to
// the next one, and that page opens a scene rather than continuing one, so it
// must not claim to continue the scene the previous page ended.
//
// The head is drawn by the page header, which runs before any of the page's
// content, so the decision has to be taken when the page is opened. The flag
// kept its value from the previous page until this was fixed, and a page that
// started a fresh scene inherited the head of the scene that had just ended.
func TestHeadingPushedToANewPageCarriesNoHead(t *testing.T) {
	action := strings.TrimSuffix(strings.Repeat("The office is empty and the light is on. ", 2), " ")
	// The leading heading is what makes the scene open, so the second one is a
	// new scene arriving inside a page that is already carrying the first.
	// One line free at the foot: not enough for two blank lines, the heading,
	// another blank and a line of its action, so the heading is pushed whole.
	src := "INT. HOUSE - DAY\n\n" + leavingFree("EXT. STREET - NIGHT\n"+action+"\n", 2, 1)

	rendered := frontMatter(true, src)
	lines := bodyLines(t, rendered)
	// The title page is page 0, so the body starts at 1 and "pushed to a new
	// page" means the heading is no longer on the first page of script.
	heading := find(t, lines, "EXT. STREET - NIGHT")
	if heading.page() < 2 {
		t.Fatalf("the heading is on page %d, want it pushed off the first page of script", heading.page()+1)
	}

	firsts := bodyPageFirst(t, rendered)
	pages := marginMarksByPage(t, rendered)
	if len(firsts) != len(pages) {
		t.Fatalf("bodyPageFirst has %d pages and marginMarksByPage %d, want the same", len(firsts), len(pages))
	}
	if got := pages[heading.page()].heads; len(got) != 0 {
		t.Errorf("the page the heading was pushed to carries %v, want no %s head: it opens a scene",
			textsOfLines(got), continuedHd)
	}
	if first := firsts[heading.page()]; !isSceneHeadingText(first) {
		t.Errorf("that page begins with %q, want the scene heading", first)
	}
}

// TestPageOpeningMidSceneKeepsItsHead is the other side of the same flag: a page
// that picks a scene up where the last one left off does carry the head.
func TestPageOpeningMidSceneKeepsItsHead(t *testing.T) {
	// A scene heading opens the scene, then a paragraph of twenty character
	// sentences, three to a line, long enough that it cannot fit on one page and
	// has to be divided rather than pushed.
	var b strings.Builder
	b.WriteString("INT. HOUSE - DAY\n\n")
	for i := 0; i < 200; i++ {
		b.WriteString("The light lies. ")
	}
	b.WriteString("\n")

	rendered := frontMatter(true, b.String())
	lines := bodyLines(t, rendered)
	pages := marginMarksByPage(t, rendered)
	if len(lines) == 0 {
		t.Fatal("no body lines, want the paragraph to be divided")
	}

	// The tail of the paragraph is the last line of script in the document, and
	// the page it lands on is the one that had to open in the middle of a scene.
	tail := lines[len(lines)-1]
	got := pages[tail.page()].heads
	if len(got) != 1 {
		t.Fatalf("the page the paragraph was carried onto carries %d heads, want 1", len(got))
	}
	if want := continuedHd + " 1"; got[0].text != want {
		t.Errorf("head = %q, want %q", got[0].text, want)
	}
}

func isSceneHeadingText(s string) bool {
	for _, p := range []string{"INT.", "EXT.", "INT./EXT.", "EST."} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// TestSceneEndsAfter is the decision behind both the (CONTINUED) foot and the
// CONTINUED: head when a page is opened between two elements.
func TestSceneEndsAfter(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// want is indexed by the element the break falls after, skipping
		// invisible ones.
		want []bool
	}{
		{
			name: "a heading starts a scene",
			src:  "INT. HOUSE - DAY\n\nA door stands open.\n",
			want: []bool{true, false},
		},
		{
			// The page holding the action is closed by a heading arriving on the
			// next one, so the scene has not carried over.
			name: "a transition leading to a heading closes the scene",
			src:  "INT. HOUSE - DAY\n\nA door stands open.\n\nCUT TO:\n\nEXT. STREET - NIGHT\n",
			want: []bool{true, true, true, true},
		},
		{
			// The case that lost the notation: a transition used as a time card
			// in the middle of a scene, followed by action.
			name: "a transition leading to action leaves the scene running",
			src:  "INT. HOUSE - DAY\n\nA door stands open.\n\n(POCO DESPUÉS) CUT TO:\n\nA door swings open.\n",
			want: []bool{true, false, false, false},
		},
		{
			// The new page holds this scene's last element, so the scene has run
			// onto it and the page it was pushed off carries the (CONTINUED).
			name: "a last element pushed to a new page continues the scene",
			src:  "INT. HOUSE - DAY\n\nA door stands open.\n",
			want: []bool{true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			els := parser.Parse(tt.src).Elements
			var got []bool
			for i, el := range els {
				if !visible(el) {
					continue
				}
				got = append(got, sceneEndsAfter(els, i))
			}
			if len(got) != len(tt.want) {
				t.Fatalf("sceneEndsAfter over %d elements = %v, want %d values", len(got), got, len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("element %d (%T %q): sceneEndsAfter = %t, want %t",
						i, els[i], elText(els[i]), got[i], tt.want[i])
				}
			}
		})
	}
}

func elText(el parser.Element) string {
	switch e := el.(type) {
	case parser.SceneHeading:
		return e.Text
	case parser.Action:
		return e.Text
	case parser.Transition:
		return e.Text
	case parser.Character:
		return e.Text
	case parser.Dialogue:
		return e.Text
	}
	return ""
}

// TestTimeCardTransitionKeepsTheNotation checks the observable effect of the rule
// above: a page that breaks after a transition used as a time card still carries
// the (CONTINUED), and the next page still opens with the head. The transition is
// written the way the author would, ending in TO:, which is what makes it a
// transition and therefore the interesting case.
func TestTimeCardTransitionKeepsTheNotation(t *testing.T) {
	// A heading opens the scene, then one line free, so the transition stays on
	// the first page and the break is taken for the action that follows it.
	src := "INT. HOUSE - DAY\n\n" +
		leavingFree("(POCO DESPUÉS) CUT TO:\nThe office is empty and the light is on.\n", 1, 1)

	pages := marginMarksByPage(t, frontMatter(true, src))
	// Page 0 is the title page, so the body starts at 1.
	if len(pages) < 2 {
		t.Fatalf("document produced %d pages, want the body to break", len(pages))
	}
	if got := len(pages[1].feet); got != 1 {
		t.Errorf("the page holding the time card carries %d (CONTINUED) feet, want 1", got)
	}
	if got := len(pages[2].heads); got != 1 {
		t.Errorf("the next page carries %d CONTINUED: heads, want 1", got)
	}
}
