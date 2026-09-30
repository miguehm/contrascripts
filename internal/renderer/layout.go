package renderer

import (
	"math"
	"strings"

	"github.com/user/fountain/internal/parser"
)

// linesPerPage is how many 12pt line boxes fit between the top and bottom
// margins on US Letter: 9.0 usable inches at six lines per inch. It is the
// figure behind the one-page-per-minute rule of thumb, so getting it wrong
// changes a script's running time.
const linesPerPage = 54

// Page-breaking thresholds, in lines. A split is only allowed when both sides
// clear their minimum, which is also what stops a short block from being torn
// in two: a three line block can never leave two behind and carry two over, so
// it is never split, and a heading above it moves down whole.
const (
	orphanLimit = 2 // minimum lines left behind on the page being left
	widowLimit  = 2 // minimum lines carried over to the next page
)

// A visualLine is one measured line box of a block, before it is placed. A blank
// line comes from an explicit break in the source text and still occupies a line
// of the grid.
type visualLine struct {
	chunks []styledChunk
	blank  bool
	// endsSentence records whether the line finished a sentence, which the
	// breaker prefers as a split point so a page never ends mid-sentence.
	endsSentence bool
}

// bodyLine is the zero-based index of the line box the cursor is about to write.
// It is derived from the cursor position so the accounting cannot drift from
// what is actually on the page; a fresh page parks the cursor on the top margin,
// which is line 0.
func (r *PDFRenderer) bodyLine() int {
	n := int(math.Round((r.pdf.GetY() - marginTop) / lineSpacing))
	if n < 0 {
		return 0
	}
	return n
}

// linesLeft is how many more line boxes fit on the current page.
func (r *PDFRenderer) linesLeft() int {
	return linesPerPage - r.bodyLine()
}

// newBodyPage starts a new page for the screenplay body. gofpdf's own auto page
// break stays enabled as a backstop, but every break in the body is decided
// here so the rules below can be applied.
// newBodyPage starts a new page for the screenplay body. sceneContinues says
// whether the scene open where the page is being left is still running past it,
// which is what puts a (CONTINUED) at the foot of the closing page and a
// CONTINUED: head at the top of the next one.
//
// The caller has to say, because only the caller knows whether the page is being
// opened between two elements, where the decision turns on what follows, or in
// the middle of a block that is being divided, where the scene plainly continues
// because its tail is what lands on the new page. Both flags are assigned
// unconditionally: leaving either to keep its previous value made a page that
// opened a fresh scene still claim to continue the one before it.
func (r *PDFRenderer) newBodyPage(sceneContinues bool) {
	r.carryOn = sceneContinues
	r.headOnNewPage = sceneContinues && !r.nextStartsScene
	r.pdf.AddPage()
	r.prevVisible = nil
}

// sceneEndsAfter reports whether the scene open at index i is finished, for a
// break taken between two elements.
//
// The page is handed to whatever comes next, so the question is simply whether
// that is a scene heading. A transition is skipped, because a transition is also
// how a time card is often written, sitting in the middle of a scene with action
// after it: treating every transition as a scene break silently dropped the
// (CONTINUED) from every page that ran past one, when the scene was still running.
//
// A block divided across a break does not come through here: its tail is what
// lands on the new page, so the scene plainly continues and the caller says so.
func sceneEndsAfter(els []parser.Element, i int) bool {
	if isSceneHeading(els[i]) {
		return true
	}
	for j := i + 1; j < len(els); j++ {
		if !visible(els[j]) || isTransition(els[j]) {
			continue
		}
		return isSceneHeading(els[j])
	}
	// Nothing after this one, so the new page holds this scene's last element and
	// the scene does run onto it.
	return false
}

// measureBlock wraps text into the visual lines it occupies at the given width,
// without drawing anything. Measuring first is what makes the page-breaking
// rules expressible: the breaker has to know a block's line count before
// committing any of it to the page.
func (r *PDFRenderer) measureBlock(text string, width float64, baseBold, baseItalic bool) []visualLine {
	chunks := mergeChunks(flattenInline(parser.ParseInline(text), baseBold, baseItalic))

	// Translate each chunk to CP1252 single-byte so gofpdf's byte-based width
	// computation matches the rendered output.
	translated := make([]styledChunk, len(chunks))
	for i, c := range chunks {
		translated[i] = styledChunk{
			text:   strings.Replace(r.tr(c.text), "\r", "", -1),
			bold:   c.bold,
			italic: c.italic,
		}
	}

	wrapped := r.wrapStyled(translated, width)
	lines := make([]visualLine, 0, len(wrapped))
	for _, line := range wrapped {
		if len(line) == 0 {
			lines = append(lines, visualLine{blank: true})
			continue
		}
		lines = append(lines, visualLine{chunks: line, endsSentence: endsSentence(line)})
	}
	return lines
}

// endsSentence reports whether the last chunk of a line finishes a sentence.
// Dialogue may also be broken at a comma, which the format allows, so a comma
// counts alongside the usual terminators.
func endsSentence(line []styledChunk) bool {
	if len(line) == 0 {
		return false
	}
	text := strings.TrimRight(line[len(line)-1].text, " \t")
	if text == "" {
		return false
	}
	switch text[len(text)-1] {
	case '.', '!', '?', ',', ';', ':':
		return true
	}
	return false
}

// measureElement returns the visual lines el occupies, or nil for an element
// that takes no space. The widths mirror the columns the format specifies.
func (r *PDFRenderer) measureElement(el parser.Element) []visualLine {
	switch e := el.(type) {
	case parser.SceneHeading:
		return r.measureBlock(strings.ToUpper(e.Text), pageWidth-charLeft-marginRight, false, false)
	case parser.Action:
		return r.measureBlock(e.Text, pageWidth-actionLeft-marginRight, false, false)
	case parser.Character:
		return r.measureBlock(strings.ToUpper(e.Text), pageWidth-charNameLeft-marginRight, false, false)
	case parser.Dialogue:
		return r.measureBlock(e.Text, dialogRight-dialogLeft, false, false)
	case parser.Parenthetical:
		return r.measureBlock(e.Text, parenRight-parenLeft, false, false)
	case parser.Transition:
		return r.measureBlock(strings.ToUpper(e.Text), pageWidth, false, false)
	case parser.Centered:
		return r.measureBlock(e.Text, pageWidth, false, false)
	case parser.Lyric:
		return r.measureBlock(e.Text, pageWidth-charLeft-marginRight, false, true)
	case parser.Forced:
		return r.measureElement(forcedAs(e))
	}
	return nil
}

// forcedAs resolves a forced element to the element it stands for, so it is
// measured and paginated exactly as if the prefix had not been typed.
func forcedAs(el parser.Forced) parser.Element {
	switch el.Prefix {
	case '.':
		return parser.SceneHeading{Text: el.Text}
	case '@':
		return parser.Character{Text: el.Text}
	default:
		return parser.Action{Text: el.Text}
	}
}

// splitPoint chooses how many of a block's lines stay on the current page. It
// returns 0 to push the whole block to the next page.
//
// A split needs at least orphanLimit lines behind and widowLimit lines ahead.
// Among the candidates that clear both, prefer one that ends a sentence;
// failing that, take the furthest one that still clears the minimums.
func splitPoint(lines []visualLine, available int) int {
	if len(lines) <= available {
		return len(lines) // the whole block fits
	}
	max := len(lines) - widowLimit
	if max > available {
		max = available
	}
	if max < orphanLimit {
		return 0 // too little would remain behind, so push the block whole
	}
	for i := max; i >= orphanLimit; i-- {
		if lines[i-1].endsSentence {
			return i
		}
	}
	return max
}

// placeBlock draws a measured block at startX, centring it when asked, and
// splits it across a page boundary when it cannot fit whole. It reports whether
// the block had to be divided, which is what the continuation notation keys off.
func (r *PDFRenderer) placeBlock(startX, width float64, lines []visualLine, center bool) bool {
	if len(lines) == 0 {
		return false
	}
	at := splitPoint(lines, r.linesLeft())
	if at == 0 && r.bodyLine() > 0 {
		// Pushed whole, so the new page opens with this block and the scene
		// continues onto it.
		r.newBodyPage(r.inScene)
		at = len(lines)
		if at > r.linesLeft() {
			at = r.linesLeft()
		}
	}
	r.drawVisualLines(startX, width, lines, 0, at, center)
	if at >= len(lines) {
		return false
	}
	// Divided, so the tail of this block is what lands on the new page.
	r.newBodyPage(r.inScene)
	r.drawVisualLines(startX, width, lines, at, len(lines), center)
	return true
}

// drawVisualLines writes lines[from:to], one line box each. It is the only place
// the body emits text, so a line can never straddle a page by accident: the
// break is always an explicit newBodyPage before the first line of a page.
func (r *PDFRenderer) drawVisualLines(startX, width float64, lines []visualLine, from, to int, center bool) {
	for i := from; i < to && i < len(lines); i++ {
		line := lines[i]
		if line.blank {
			r.blankLine()
			continue
		}
		if center {
			total := 0.0
			for _, seg := range line.chunks {
				r.pdf.SetFont(fontFamily, seg.style(), fontSize)
				total += r.pdf.GetStringWidth(seg.text)
			}
			r.pdf.SetX(startX + (width-total)/2)
		} else {
			r.pdf.SetX(startX)
		}
		for _, seg := range line.chunks {
			r.pdf.SetFont(fontFamily, seg.style(), fontSize)
			r.pdf.CellFormat(r.pdf.GetStringWidth(seg.text), lineSpacing, seg.text, "", 0, "L", false, 0, "")
		}
		r.blankLine()
	}
}

// keepWithNext returns how many lines beyond el itself must share its page.
// This is what stops a heading, a character cue or a parenthetical being
// stranded at the foot of a page. The blank lines introducing the next element
// count against the page as well, so they are included.
func keepWithNext(els []parser.Element, i int) int {
	next := nextVisible(els, i+1)
	if next < 0 {
		return 0
	}
	gap := spaceAbove(els[i], els[next])
	switch {
	case isCue(els[i]):
		// A cue must be followed by at least two lines of its own speech,
		// counting a parenthetical.
		return gap + orphanLimit
	case isParenthetical(els[i]):
		// Dialogue must follow, so a page may not end on a parenthetical.
		return gap + 1
	case !isSceneHeading(els[i]), isStandaloneShot(els[i]), isSceneHeading(els[next]):
		// Ending a page on a heading is the one documented exception, and it is
		// allowed only when another heading or shot follows, as with an
		// establishing shot introducing the scene it sets up.
		return 0
	}
	return gap + 1
}

func isCue(el parser.Element) bool {
	_, ok := el.(parser.Character)
	return ok
}

func isParenthetical(el parser.Element) bool {
	_, ok := el.(parser.Parenthetical)
	return ok
}

// isStandaloneShot reports whether a scene heading is a setup that legitimately
// carries no content of its own, and so may end a page alone.
func isStandaloneShot(el parser.Element) bool {
	text := strings.ToUpper(headingText(el))
	return strings.HasPrefix(text, "ESTABLISHING") || strings.HasPrefix(text, "INSERT")
}

func headingText(el parser.Element) string {
	switch e := el.(type) {
	case parser.SceneHeading:
		return e.Text
	case parser.Forced:
		return e.Text
	}
	return ""
}

// nextVisible returns the index of the first visible element at or after i, or
// -1 when there is none.
func nextVisible(els []parser.Element, i int) int {
	for ; i < len(els); i++ {
		if visible(els[i]) {
			return i
		}
	}
	return -1
}

// breakForKeepTogether decides whether the element at index i has to move to a
// new page. It runs before that element and the blank line introducing it are
// emitted, so a break never leaves a stray gap at the foot of the old page.
func (r *PDFRenderer) breakForKeepTogether(els []parser.Element, i int) {
	if r.bodyLine() == 0 {
		return // already at the top, so there is nothing to gain
	}
	el := els[i]
	lines := r.measureElement(el)
	// The gap is emitted before the element's first line, so it is part of what
	// has to fit here, and a break has to be taken before it is emitted too.
	avail := r.linesLeft() - spaceAbove(r.prevVisible, el)
	need := keepWithNext(els, i)

	// A transition may never start a page. Reserving its line here, rather than
	// checking when the transition is reached, is what makes the rule
	// enforceable at all: by then the previous block is already drawn.
	if next := nextVisible(els, i+1); next >= 0 && isTransition(els[next]) {
		if want := spaceAbove(el, els[next]) + 1; want > need {
			need = want
		}
	}

	// The element fits whole, so only the content it has to keep company with
	// can force a break.
	// A break here is taken between two elements, so whether the scene runs on
	// turns on what follows rather than on this block, which is not divided.
	runsOn := r.inScene && !sceneEndsAfter(els, i)

	if len(lines) <= avail {
		if need <= avail-len(lines) {
			return
		}
		r.newBodyPage(runsOn)
		return
	}

	// It does not fit whole, but a block that can be divided with a legal
	// fragment on each side is better off divided than pushed, and placeBlock
	// will do the dividing. Treating "does not fit" as "must move whole" here
	// is what used to strand a scene heading with the action it introduces
	// pushed over to the next page.
	if at := splitPoint(lines, avail); at > 0 && need <= at {
		return
	}
	r.newBodyPage(runsOn)
}
