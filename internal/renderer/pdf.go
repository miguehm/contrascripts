package renderer

import (
	"fmt"
	"io"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/user/fountain/internal/parser"
)

const (
	pageWidth  = 8.5
	pageHeight = 11.0

	marginLeft   = 1.5
	marginRight  = 1.0
	marginTop    = 1.0
	marginBottom = 1.0

	charLeft     = 1.5
	actionLeft   = 1.5
	charNameLeft = 3.7
	parenLeft    = 3.1
	parenRight   = 5.0
	dialogLeft   = 2.5
	dialogRight  = 6.0

	fontSize = 12.0

	// lineSpacing is the leading between consecutive baselines: exactly one
	// 12pt line, i.e. 1/6 inch, the industry rule of 6 lines per vertical
	// inch. Every block gap is expressed as a multiple of it so the whole
	// page stays on one line grid.
	lineSpacing = 1.0 / 6.0
)

type PDFRenderer struct {
	pdf         *gofpdf.Fpdf
	doc         *parser.Fountain
	tr          func(string) string
	sceneNum    int
	prevVisible parser.Element

	// Scene tracking, used by the production continuation notation. A scene is
	// open from its heading until a new heading arrives, running straight through
	// the dialogue and the transitions inside it.
	//
	// sceneOrdinal counts every scene from 1 whether or not they are numbered in
	// the margins, because the CONTINUED: head refers to it. Counting it
	// independently means turning scene-numbers on or off does not change what
	// the production notation says.
	sceneOrdinal int
	inScene      bool
	// carryOn marks the page being closed as one whose scene runs past its foot,
	// which is what the footer turns into a (CONTINUED).
	carryOn bool
	// headOnNewPage records whether the page just opened continues a scene that
	// began on an earlier one, decided when the page is opened.
	headOnNewPage bool
	// nextStartsScene says whether the element about to be drawn is a scene
	// heading, which is what newBodyPage consults to keep a fresh scene off a
	// page that should not claim to continue the previous one.
	nextStartsScene bool
	// lastCue is the cue most recently drawn, which is the one to repeat when a
	// speech continues over a page break.
	lastCue *parser.Character
}

func New(doc *parser.Fountain) *PDFRenderer {
	pdf := gofpdf.New("P", "in", "Letter", "")
	pdf.SetMargins(marginLeft, marginTop, marginRight)
	pdf.SetAutoPageBreak(true, marginBottom)
	// gofpdf pads every cell by a 1mm cell margin by default. On a
	// monospaced screenplay grid that padding pushes text off its column and
	// steals two millimetres from the wrap width, which is enough to break a
	// full 60-character action line onto a second line. The cell origin is
	// already the column, so the margin has to go.
	pdf.SetCellMargin(0)
	registerCourierPrime(pdf)
	if pdf.Err() {
		pdf.SetFont("Courier", "", fontSize)
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		return &PDFRenderer{
			pdf: pdf,
			doc: doc,
			tr:  tr,
		}
	}
	pdf.SetFont(fontFamily, "", fontSize)
	tr := func(s string) string { return s }

	return &PDFRenderer{
		pdf: pdf,
		doc: doc,
		tr:  tr,
	}
}

func (r *PDFRenderer) RenderToFile(path string) error {
	r.render()
	return r.pdf.OutputFileAndClose(path)
}

func (r *PDFRenderer) RenderToWriter(w io.Writer) error {
	r.render()
	return r.pdf.Output(w)
}

func (r *PDFRenderer) render() {
	// The header runs as a page opens, which is the only band of the page that is
	// free of body text. The continuation marks live here rather than in the text
	// block so that they cannot collide with the first line of the page and do
	// not consume any of its 54 lines.
	r.pdf.SetHeaderFuncMode(func() {
		if r.production() && r.headOnNewPage {
			r.drawContinuedHead(r.sceneOrdinal)
		}
		if r.pdf.PageNo() > 1 {
			// 0.5in from the top, flush to the 7.5in right margin, and absent
			// from the first page. The trailing period is the convention in the
			// screenplay format guides; Final Draft's own default omits it.
			r.pdf.SetFont(fontFamily, "", 10)
			r.pdf.SetXY(pageWidth-marginRight-0.5, 0.5)
			r.pdf.CellFormat(0.5, 0.25, fmt.Sprintf("%d.", r.pdf.PageNo()-1), "", 0, "R", false, 0, "")
		}
	}, true)

	// The footer runs as a page closes, which is the only moment the (CONTINUED)
	// mark can still be placed on it.
	r.pdf.SetFooterFunc(func() {
		if r.production() && r.carryOn {
			r.drawContinued()
		}
		r.carryOn = false
	})

	if hasTitlePage(r.doc) {
		r.renderTitlePage()
	}

	r.renderElements()
}

func hasTitlePage(doc *parser.Fountain) bool {
	tp := doc.TitlePage
	return tp.Title != "" || tp.Credit != "" || tp.Author != "" ||
		tp.DraftDate != "" || tp.Contact != "" || tp.Source != "" ||
		tp.Revision != "" || tp.Logline != "" || tp.Tagline != "" ||
		tp.Synopsis != "" || tp.Genre != "" || tp.Subgenre != "" ||
		tp.Tone != "" || tp.Themes != "" || tp.Comps != "" ||
		tp.Format != "" || tp.Season != "" || tp.Episode != "" ||
		tp.EpisodeTitle != "" || tp.Runtime != "" || tp.Pages != "" ||
		tp.Language != "" || tp.Status != "" || tp.ProdCompany != "" ||
		tp.Producer != "" || tp.Director != "" || tp.Budget != "" ||
		tp.AdaptedBy != "" || tp.BasedOn != "" || tp.Copyright != "" ||
		tp.WGA != "" || tp.Rights != "" || tp.Dedication != "" ||
		tp.Disclaimer != "" || tp.DraftType != "" || tp.Version != "" ||
		tp.RevisionColor != "" || tp.RevisionDate != "" || tp.Email != "" ||
		tp.Phone != "" || tp.Agent != "" || tp.Manager != "" ||
		tp.Address != "" || len(tp.Custom) > 0
}

func (r *PDFRenderer) renderTitlePage() {
	r.pdf.AddPage()
	tp := r.doc.TitlePage

	// center writes a (possibly multi-line) field centered across the page.
	center := func(text string) {
		if text == "" {
			return
		}
		r.pdf.SetLeftMargin(0)
		r.pdf.SetRightMargin(0)
		r.pdf.SetX(0)
		r.pdf.MultiCell(pageWidth, lineSpacing, r.tr(text), "", "C", false)
	}
	centerStyled := func(text, style string) {
		if text == "" {
			return
		}
		r.pdf.SetFont(fontFamily, style, fontSize)
		center(text)
		r.pdf.SetFont(fontFamily, "", fontSize)
	}
	gap := func(h float64) {
		r.pdf.SetY(r.pdf.GetY() + h)
	}
	metaLine := func(parts ...string) {
		var keep []string
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				keep = append(keep, p)
			}
		}
		if len(keep) == 0 {
			return
		}
		center(strings.Join(keep, "  |  "))
	}

	r.pdf.SetFont(fontFamily, "", fontSize)

	// Zone 1: identity top-center.
	if tp.Title != "" {
		r.pdf.SetY(3.0)
		r.renderTitleCentered(tp.Title, center)
		gap(0.4)
	}
	if tp.Credit != "" {
		center(tp.Credit)
		gap(0.2)
	}
	if tp.Author != "" {
		center(tp.Author)
		gap(0.3)
	}
	if tp.AdaptedBy != "" {
		center(tp.AdaptedBy)
		gap(0.2)
	}
	if tp.BasedOn != "" {
		center(tp.BasedOn)
		gap(0.2)
	}
	if tp.EpisodeTitle != "" {
		centerStyled("\""+tp.EpisodeTitle+"\"", "I")
		gap(0.2)
	}

	// Zone 2: editorial middle-center.
	if tp.Logline != "" {
		centerStyled(tp.Logline, "I")
		gap(0.2)
	}
	if tp.Tagline != "" {
		center(tp.Tagline)
		gap(0.2)
	}
	metaLine(tp.Genre, tp.Subgenre, tp.Tone)
	if tp.Genre != "" || tp.Subgenre != "" || tp.Tone != "" {
		gap(0.15)
	}
	metaLine(tp.Format, seasonEpisodeText(tp), tp.Runtime, tp.Language)
	if tp.Format != "" || tp.Season != "" || tp.Episode != "" || tp.Runtime != "" || tp.Language != "" {
		gap(0.15)
	}
	if tp.Comps != "" {
		center("Comps: " + tp.Comps)
		gap(0.15)
	}
	if tp.Themes != "" {
		center("Themes: " + tp.Themes)
		gap(0.15)
	}
	if tp.Status != "" || tp.ProdCompany != "" || tp.Producer != "" || tp.Director != "" || tp.Budget != "" {
		gap(0.1)
		metaLine(tp.Status, tp.ProdCompany)
		metaLine(tp.Producer, tp.Director, tp.Budget)
		gap(0.15)
	}

	if tp.Source != "" {
		center(tp.Source)
		gap(0.2)
	}
	if tp.Revision != "" {
		center(tp.Revision)
		gap(0.2)
	}
	for k, v := range tp.Custom {
		if v != "" {
			center(fmt.Sprintf("%s: %s", k, v))
			gap(0.15)
		}
	}

	// Zone 3: version + contact footer.
	draftLeft := strings.Join(nonEmpty(tp.DraftType, tp.Version, tp.RevisionColor, tp.RevisionDate, tp.DraftDate, tp.Pages), "  ")
	contactRight := strings.Join(nonEmpty(tp.Contact, tp.Address, tp.Email, tp.Phone, tp.Agent, tp.Manager), "\n")
	legal := strings.Join(nonEmpty(tp.Copyright, tp.WGA, tp.Rights), "\n")
	if draftLeft != "" || contactRight != "" || legal != "" {
		const bottomY = 8.0
		if r.pdf.GetY() < bottomY {
			r.pdf.SetY(bottomY)
		} else {
			gap(0.3)
		}
		if draftLeft != "" {
			center(draftLeft)
			gap(0.2)
		}
		if contactRight != "" {
			center(contactRight)
			gap(0.2)
		}
		if legal != "" {
			center(legal)
			gap(0.15)
		}
	}

	// Overflow: synopsis, dedication, disclaimer go to page 2.
	if tp.Synopsis != "" || tp.Dedication != "" || tp.Disclaimer != "" {
		r.pdf.AddPage()
		r.pdf.SetLeftMargin(marginLeft)
		r.pdf.SetRightMargin(marginRight)
		if tp.Synopsis != "" {
			r.pdf.SetX(marginLeft)
			r.pdf.MultiCell(0, lineSpacing, r.tr("Synopsis: "+tp.Synopsis), "", "L", false)
			gap(0.2)
		}
		if tp.Dedication != "" {
			r.pdf.SetX(marginLeft)
			r.pdf.MultiCell(0, lineSpacing, r.tr(tp.Dedication), "", "C", false)
			gap(0.2)
		}
		if tp.Disclaimer != "" {
			r.pdf.SetX(marginLeft)
			r.pdf.MultiCell(0, lineSpacing, r.tr(tp.Disclaimer), "", "L", false)
		}
	}

	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderTitleCentered(raw string, center func(string)) {
	upper := strings.ToUpper(raw)
	inlines := parser.ParseInline(upper)
	chunks := mergeChunks(flattenInline(inlines, false, false))
	if !hasFormatting(chunks, false, false) {
		center(upper)
		return
	}
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	translated := make([]styledChunk, len(chunks))
	for i, c := range chunks {
		translated[i] = styledChunk{text: strings.Replace(r.tr(c.text), "\r", "", -1), bold: c.bold, italic: c.italic}
	}
	lines := r.wrapStyled(translated, pageWidth)
	for _, line := range lines {
		if line == nil {
			r.pdf.Ln(lineSpacing)
			continue
		}
		var totalW float64
		for _, seg := range line {
			r.pdf.SetFont(fontFamily, seg.style(), fontSize)
			totalW += r.pdf.GetStringWidth(seg.text)
		}
		r.pdf.SetX((pageWidth - totalW) / 2.0)
		for _, seg := range line {
			r.pdf.SetFont(fontFamily, seg.style(), fontSize)
			w := r.pdf.GetStringWidth(seg.text)
			r.pdf.CellFormat(w, lineSpacing, seg.text, "", 0, "L", false, 0, "")
		}
		r.pdf.Ln(lineSpacing)
	}
	r.pdf.SetFont(fontFamily, "", fontSize)
}

func seasonEpisodeText(tp parser.TitlePage) string {
	var se string
	if tp.Season != "" && tp.Episode != "" {
		se = "S" + tp.Season + "E" + tp.Episode
	} else if tp.Season != "" {
		se = "Season " + tp.Season
	} else if tp.Episode != "" {
		se = "Episode " + tp.Episode
	}
	return se
}

func nonEmpty(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// blankLine advances the cursor by exactly one screenplay line, so the next
// block starts one blank line below the previous one. Because gofpdf's Ln also
// resets the cursor to the left margin, callers must set their own margin and X
// afterwards, which every renderer here does.
func (r *PDFRenderer) blankLine() {
	r.pdf.Ln(lineSpacing)
}

// spaceAbove is the number of blank lines the screenplay format puts between
// the previous element and el. The rules are pairwise, because a gap depends on
// both sides: a scene heading is separated from whatever precedes it by two
// blank lines so consecutive scenes read as distinct beats, a transition is
// followed by two, and a character / parenthetical / dialogue group stays
// single spaced throughout. A nil prev means el starts a page, where the top
// margin is already the boundary.
func spaceAbove(prev, el parser.Element) int {
	switch {
	case prev == nil:
		return 0
	case isDialogueGroup(prev) && isDialogueGroup(el):
		// Inside the cue, parenthetical and dialogue run there is no gap at
		// all. Entering or leaving the run still gets the usual blank line.
		return 0
	case isSceneHeading(el):
		return 2
	case isTransition(prev):
		return 2
	default:
		return 1
	}
}

func isSceneHeading(el parser.Element) bool {
	switch e := el.(type) {
	case parser.SceneHeading:
		return true
	case parser.Forced:
		return e.Prefix == '.'
	}
	return false
}

func isTransition(el parser.Element) bool {
	_, ok := el.(parser.Transition)
	return ok
}

// isDialogueGroup reports whether el is one of the three elements that share a
// single-spaced run: the character cue, its parenthetical and its dialogue.
func isDialogueGroup(el parser.Element) bool {
	switch el.(type) {
	case parser.Character, parser.Parenthetical, parser.Dialogue:
		return true
	}
	return false
}

// visible reports whether el puts anything on the page. Notes and boneyards are
// parsed but never printed, so they must not consume the blank line the next
// visible element expects.
func visible(el parser.Element) bool {
	switch el.(type) {
	case parser.Note, parser.Boneyard:
		return false
	}
	return true
}

func (r *PDFRenderer) renderElements() {
	r.pdf.AddPage()
	r.sceneNum = 0
	r.prevVisible = nil

	for i, el := range r.doc.Elements {
		if !visible(el) {
			continue
		}
		// What the upcoming element means for the notation is published before the
		// break is taken, because a page that opens with a scene heading starts a
		// scene rather than continuing one, and both facts are only knowable here.
		r.nextStartsScene = isSceneHeading(el)
		if isSceneHeading(el) {
			r.inScene = true
		}
		r.breakForKeepTogether(r.doc.Elements, i)
		r.renderElement(r.doc.Elements, i)
	}
}

func (r *PDFRenderer) renderElement(els []parser.Element, i int) {
	el := els[i]
	if !visible(el) {
		return
	}

	for i := 0; i < spaceAbove(r.prevVisible, el); i++ {
		r.blankLine()
	}

	switch e := el.(type) {
	case parser.SceneHeading:
		r.renderSceneHeading(e)
	case parser.Action:
		r.renderAction(e)
	case parser.Character:
		r.renderCharacter(e)
	case parser.Dialogue:
		r.renderDialogue(e)
	case parser.Parenthetical:
		r.renderParenthetical(e)
	case parser.Transition:
		r.renderTransition(e)
	case parser.Centered:
		r.renderCentered(e)
	case parser.Lyric:
		r.renderLyric(e)
	case parser.Forced:
		r.renderForced(e)
	case parser.PageBreak:
		// A forced break resets the run, so the next element starts on the
		// next page with no separation from the margin.
		r.renderPageBreak(els, i)
		r.prevVisible = nil
		return
	}

	r.prevVisible = el
}

func (r *PDFRenderer) renderSceneHeading(el parser.SceneHeading) {
	// Every scene is counted, so the CONTINUED: head can name it even when the
	// scene numbers are not being drawn in the margins.
	r.sceneOrdinal++
	showNum := r.doc.TitlePage.SceneNumbers
	num := 0
	if showNum {
		r.sceneNum++
		num = r.sceneNum
	}
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	startY := r.pdf.GetY()
	r.placeBlock(charLeft, pageWidth-charLeft-marginRight,
		r.measureBlock(strings.ToUpper(el.Text), pageWidth-charLeft-marginRight, false, false), false)
	if showNum {
		r.drawSceneNumber(num, startY)
	}
	r.pdf.SetFont(fontFamily, "", fontSize)
}

// drawSceneNumber writes the scene number in both page margins at the given
// Y without touching the heading text block. The cells advance the cursor, so
// the caller restores it.
func (r *PDFRenderer) drawSceneNumber(num int, y float64) {
	label := r.tr(fmt.Sprintf("%d", num))
	r.pdf.SetFont(fontFamily, "", fontSize)
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	r.pdf.SetXY(0.75, y)
	r.pdf.CellFormat(0.5, lineSpacing, label, "", 0, "R", false, 0, "")
	r.pdf.SetXY(pageWidth-marginRight+0.15, y)
	r.pdf.CellFormat(0.5, lineSpacing, label, "", 0, "L", false, 0, "")
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetY(y + lineSpacing)
}

func (r *PDFRenderer) renderAction(el parser.Action) {
	r.pdf.SetLeftMargin(actionLeft)
	r.pdf.SetRightMargin(marginRight)
	r.placeBlock(actionLeft, pageWidth-actionLeft-marginRight,
		r.measureElement(el), false)
}

func (r *PDFRenderer) renderCharacter(el parser.Character) {
	r.pdf.SetLeftMargin(charNameLeft)
	r.pdf.SetRightMargin(marginRight)
	r.lastCue = &parser.Character{Text: el.Text}
	r.placeBlock(charNameLeft, pageWidth-charNameLeft-marginRight,
		r.measureElement(el), false)
	r.pdf.SetFont(fontFamily, "", fontSize)
}

// renderDialogue owns the division of a speech, because a divided speech is the
// one place the production notation has to be interleaved with the text: (MORE)
// at the foot of the first page and the cue again, with (CONT'D), above the
// remainder. Splitting it here rather than inside placeBlock is what keeps those
// two marks on the right lines of the right pages.
func (r *PDFRenderer) renderDialogue(el parser.Dialogue) {
	lines := r.measureElement(el)
	r.pdf.SetLeftMargin(dialogLeft)
	r.pdf.SetRightMargin(pageWidth - dialogRight)

	// A speech too tall for the space left moves whole to the next page before
	// being divided, so its first half always begins a page.
	if r.bodyLine() > 0 && r.speechSplit(lines, 0) == 0 {
		// Pushed whole to the next page, which the speech then opens.
		r.newBodyPage(r.inScene)
	}

	from := 0
	// The bound only matters for a speech taller than a whole page, which cannot
	// happen once each pass either draws lines or starts a page.
	for pass := 0; pass <= len(lines); pass++ {
		at := r.speechSplit(lines, from)
		if at == 0 {
			r.newBodyPage(r.inScene)
			continue
		}
		r.drawVisualLines(dialogLeft, dialogRight-dialogLeft, lines, from, from+at, false)
		if from+at >= len(lines) {
			return
		}
		if r.production() {
			r.drawMore()
		}
		// Divided, so the rest of the speech is what lands on the new page.
		r.newBodyPage(r.inScene)
		if r.production() {
			if cue := r.openCue(); cue != nil {
				r.drawContinuedCue(*cue)
			}
		}
		from += at
	}
}

// speechSplit is splitPoint with the space a continuation marker needs reserved.
// A (MORE) is a line of the grid in its own right, so a page that is about to
// hand a speech over to the next one has to leave room for it.
func (r *PDFRenderer) speechSplit(lines []visualLine, from int) int {
	available := r.linesLeft()
	if r.production() && from+available < len(lines) {
		available--
	}
	return splitPoint(lines[from:], available)
}

// openCue returns the cue of the group the renderer is inside, so a divided
// speech knows which name to repeat. The renderer does not look backwards
// through the document, so the cue is remembered as it is drawn.
func (r *PDFRenderer) openCue() *parser.Character { return r.lastCue }

func (r *PDFRenderer) renderParenthetical(el parser.Parenthetical) {
	r.pdf.SetLeftMargin(parenLeft)
	r.pdf.SetRightMargin(pageWidth - parenRight)
	r.placeBlock(parenLeft, parenRight-parenLeft, r.measureElement(el), false)
}

func (r *PDFRenderer) renderTransition(el parser.Transition) {
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	lines := r.measureElement(el)
	if len(lines) > 0 && !lines[0].blank {
		// Transitions are flush right, so the line is measured and then pushed
		// over by the width of its own text.
		total := 0.0
		for _, seg := range lines[0].chunks {
			r.pdf.SetFont(fontFamily, seg.style(), fontSize)
			total += r.pdf.GetStringWidth(seg.text)
		}
		r.placeBlock(pageWidth-marginRight-total, total, lines, false)
	}
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderCentered(el parser.Centered) {
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	r.placeBlock(0, pageWidth, r.measureElement(el), true)
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderLyric(el parser.Lyric) {
	r.pdf.SetFont(fontFamily, "I", fontSize)
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	r.placeBlock(charLeft, pageWidth-charLeft-marginRight, r.measureElement(el), false)
	r.pdf.SetFont(fontFamily, "", fontSize)
}

func (r *PDFRenderer) renderForced(el parser.Forced) {
	switch el.Prefix {
	case '.':
		r.renderSceneHeading(parser.SceneHeading{Text: el.Text})
	case '!':
		r.renderAction(parser.Action{Text: el.Text})
	case '@':
		r.renderCharacter(parser.Character{Text: el.Text})
	default:
		r.renderAction(parser.Action{Text: el.Text})
	}
}

func (r *PDFRenderer) renderPageBreak(els []parser.Element, i int) {
	// A forced break inside a scene is a scene running on like any other, so it
	// goes through the same path and picks up the same notation. What follows the
	// break decides, as it does for any other break between two elements.
	runsOn := r.inScene
	if j := nextVisible(els, i+1); j < 0 || isSceneHeading(els[j]) {
		runsOn = false
	}
	r.newBodyPage(runsOn)
}

// styledChunk represents a run of text with a specific font style.
type styledChunk struct {
	text   string
	bold   bool
	italic bool
}

func (c styledChunk) style() string {
	switch {
	case c.bold && c.italic:
		return "BI"
	case c.bold:
		return "B"
	case c.italic:
		return "I"
	default:
		return ""
	}
}

// flattenInline converts an InlineElement tree into a flat list of styledChunks.
func flattenInline(elements []parser.InlineElement, bold, italic bool) []styledChunk {
	var chunks []styledChunk
	for _, el := range elements {
		switch e := el.(type) {
		case parser.PlainText:
			chunks = append(chunks, styledChunk{text: e.Text, bold: bold, italic: italic})
		case parser.Bold:
			chunks = append(chunks, flattenInline(e.Content, true, italic)...)
		case parser.Italic:
			chunks = append(chunks, flattenInline(e.Content, bold, true)...)
		}
	}
	return chunks
}

// mergeChunks combines adjacent chunks with the same style.
func mergeChunks(chunks []styledChunk) []styledChunk {
	if len(chunks) == 0 {
		return chunks
	}
	merged := []styledChunk{chunks[0]}
	for i := 1; i < len(chunks); i++ {
		last := &merged[len(merged)-1]
		if last.bold == chunks[i].bold && last.italic == chunks[i].italic {
			last.text += chunks[i].text
		} else {
			merged = append(merged, chunks[i])
		}
	}
	return merged
}

// hasFormatting returns true if any chunk has bold or italic formatting
// that differs from the base style.
func hasFormatting(chunks []styledChunk, baseBold, baseItalic bool) bool {
	for _, c := range chunks {
		if c.bold != baseBold || c.italic != baseItalic {
			return true
		}
	}
	return false
}

// wrapStyled wraps translated styled chunks into lines using greedy word-wrap.
// Each chunk's text is tokenized into words, spaces, and newlines. Lines are
// built greedily, breaking before the word that would exceed the width. Explicit
// newlines produce hard line breaks.
func (r *PDFRenderer) wrapStyled(chunks []styledChunk, width float64) [][]styledChunk {
	type token struct {
		text      string
		bold      bool
		italic    bool
		isNewline bool
	}
	var tokens []token
	for _, c := range chunks {
		i := 0
		for i < len(c.text) {
			b := c.text[i]
			if b == '\n' {
				tokens = append(tokens, token{isNewline: true, bold: c.bold, italic: c.italic})
				i++
			} else if b == ' ' {
				j := i
				for j < len(c.text) && c.text[j] == ' ' {
					j++
				}
				tokens = append(tokens, token{text: c.text[i:j], bold: c.bold, italic: c.italic})
				i = j
			} else {
				j := i
				for j < len(c.text) && c.text[j] != ' ' && c.text[j] != '\n' {
					j++
				}
				tokens = append(tokens, token{text: c.text[i:j], bold: c.bold, italic: c.italic})
				i = j
			}
		}
	}

	var lines [][]styledChunk
	var cur []styledChunk
	curW := 0.0
	for _, tok := range tokens {
		if tok.isNewline {
			lines = append(lines, cur)
			cur = nil
			curW = 0
			continue
		}
		style := styledChunk{bold: tok.bold, italic: tok.italic}
		sw := r.stringWidth(tok.text, tok.bold, tok.italic)
		if curW+sw > width && len(cur) > 0 {
			lines = append(lines, cur)
			cur = nil
			curW = 0
		}
		// A token too wide for a whole empty line, such as an unbroken run of
		// characters, still has to wrap. gofpdf splits these on its own wrap
		// path, so skipping it here would let the text run into the next
		// column and out of the 60 characters the action block is defined to
		// hold.
		for sw > width {
			take := longestPrefix(r, tok.text, width, tok.bold, tok.italic)
			if take == 0 {
				break // a single glyph is wider than the column; let it overflow
			}
			lines = append(lines, append(cur, style.withText(tok.text[:take])))
			cur = nil
			curW = 0
			tok.text = tok.text[take:]
			sw = r.stringWidth(tok.text, tok.bold, tok.italic)
		}
		cur = append(cur, style.withText(tok.text))
		curW += sw
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}
	return lines
}

// withText returns a copy of the chunk carrying different text, so a forced
// split can reuse the original run's style.
func (c styledChunk) withText(text string) styledChunk {
	c.text = text
	return c
}

// longestPrefix returns the length in bytes of the longest prefix of s that
// fits within width, measured on rune boundaries so the result can never cut a
// multi-byte character in half. It returns 0 when not even the first rune fits.
//
// The boundaries come from ranging over the string, which yields the start of
// each rune, so the previous boundary is remembered rather than a byte length
// added: len(string(s[i])) is 1 for a byte even when that byte is half of a
// two-byte rune, and adding it walks the index into the middle of characters.
func longestPrefix(r *PDFRenderer, s string, width float64, bold, italic bool) int {
	best := 0
	for i := range s {
		if i == 0 {
			continue
		}
		if r.stringWidth(s[:i], bold, italic) > width {
			return best
		}
		best = i
	}
	if r.stringWidth(s, bold, italic) > width {
		return best
	}
	return len(s)
}

func (r *PDFRenderer) stringWidth(text string, bold, italic bool) float64 {
	r.pdf.SetFont(fontFamily, styledChunk{bold: bold, italic: italic}.style(), fontSize)
	return r.pdf.GetStringWidth(text)
}

func (r *PDFRenderer) styleStr(bold, italic bool) string {
	switch {
	case bold && italic:
		return "BI"
	case bold:
		return "B"
	case italic:
		return "I"
	default:
		return ""
	}
}
