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

	usableWidth = pageWidth - marginLeft - marginRight // 6.0

	charLeft     = 1.5
	charRight    = 7.5
	actionLeft   = 1.5
	actionRight  = 7.5
	charNameLeft = 3.7
	parenLeft    = 3.1
	parenRight   = 5.0
	dialogLeft   = 2.5
	dialogRight  = 6.0
	transRight   = 1.0

	fontSize    = 12.0
	lineSpacing = 0.1667 // inches per line: 12pt Courier leading (1/6 inch per screenplay line)
)

type PDFRenderer struct {
	pdf *gofpdf.Fpdf
	doc *parser.Fountain
	tr  func(string) string
}

func New(doc *parser.Fountain) *PDFRenderer {
	pdf := gofpdf.New("P", "in", "Letter", "")
	pdf.SetMargins(marginLeft, marginTop, marginRight)
	pdf.SetAutoPageBreak(true, marginBottom)
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
	r.pdf.SetHeaderFuncMode(func() {
		if r.pdf.PageNo() > 1 {
			r.pdf.SetFont(fontFamily, "", 10)
			r.pdf.SetXY(pageWidth-marginRight-0.5, 0.5)
			r.pdf.CellFormat(0.5, 0.25, fmt.Sprintf("%d", r.pdf.PageNo()-1), "", 0, "R", false, 0, "")
		}
	}, true)

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
		center(strings.ToUpper(tp.Title))
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

func (r *PDFRenderer) renderElements() {
	r.pdf.AddPage()

	for _, el := range r.doc.Elements {
		r.renderElement(el)
	}
}

func (r *PDFRenderer) renderElement(el parser.Element) {
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
		r.renderPageBreak()
	case parser.Note, parser.Boneyard:
		// invisible in output
	}
}

func (r *PDFRenderer) renderSceneHeading(el parser.SceneHeading) {
	r.pdf.Ln(0.15)
	r.pdf.SetFont(fontFamily, "", fontSize)
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charLeft)
	r.pdf.MultiCell(0, lineSpacing, r.tr(strings.ToUpper(el.Text)), "", "L", false)
	r.pdf.Ln(0.15)
	r.pdf.SetFont(fontFamily, "", fontSize)
}

func (r *PDFRenderer) renderAction(el parser.Action) {
	r.pdf.Ln(0.05)
	r.pdf.SetFont(fontFamily, "", fontSize)
	r.pdf.SetLeftMargin(actionLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(actionLeft)
	r.renderFormattedBlock(el.Text, actionLeft, pageWidth-actionLeft-marginRight, false, false)
	r.pdf.Ln(0.05)
}

func (r *PDFRenderer) renderCharacter(el parser.Character) {
	r.pdf.Ln(0.1)
	r.pdf.SetFont(fontFamily, "", fontSize)
	r.pdf.SetLeftMargin(charNameLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charNameLeft)

	name := strings.ToUpper(el.Text)

	r.pdf.MultiCell(0, lineSpacing, r.tr(name), "", "L", false)
	r.pdf.SetFont(fontFamily, "", fontSize)
}

func (r *PDFRenderer) renderDialogue(el parser.Dialogue) {
	r.pdf.SetLeftMargin(dialogLeft)
	r.pdf.SetRightMargin(pageWidth - dialogRight)
	r.pdf.SetX(dialogLeft)
	r.renderFormattedBlock(el.Text, dialogLeft, dialogRight-dialogLeft, false, false)
	r.pdf.Ln(0.05)
}

func (r *PDFRenderer) renderParenthetical(el parser.Parenthetical) {
	r.pdf.SetLeftMargin(parenLeft)
	r.pdf.SetRightMargin(pageWidth - parenRight)
	r.pdf.SetX(parenLeft)
	r.renderFormattedBlock(el.Text, parenLeft, parenRight-parenLeft, false, false)
}

func (r *PDFRenderer) renderTransition(el parser.Transition) {
	r.pdf.Ln(0.1)
	r.pdf.SetFont(fontFamily, "", fontSize)
	text := r.tr(strings.ToUpper(el.Text))
	w := r.pdf.GetStringWidth(text)
	x := pageWidth - marginRight - w
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	r.pdf.SetX(x)
	r.pdf.CellFormat(w, lineSpacing, text, "", 1, "L", false, 0, "")
	r.pdf.Ln(0.1)
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderCentered(el parser.Centered) {
	r.pdf.Ln(0.05)
	r.pdf.SetFont(fontFamily, "", fontSize)
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	r.pdf.SetX(0)

	inlines := parser.ParseInline(el.Text)
	chunks := mergeChunks(flattenInline(inlines, false, false))

	if !hasFormatting(chunks, false, false) {
		r.pdf.CellFormat(pageWidth, lineSpacing, r.tr(el.Text), "", 1, "C", false, 0, "")
	} else {
		var plainTrText string
		type centerRun struct {
			text  string
			style string
		}
		var runs []centerRun
		for _, c := range chunks {
			trText := r.tr(c.text)
			plainTrText += trText
			runs = append(runs, centerRun{
				text:  trText,
				style: c.style(),
			})
		}
		totalW := r.pdf.GetStringWidth(plainTrText)
		startX := (pageWidth - totalW) / 2.0
		r.pdf.SetX(startX)
		for _, run := range runs {
			r.pdf.SetFont(fontFamily, run.style, fontSize)
			runW := r.pdf.GetStringWidth(run.text)
			r.pdf.CellFormat(runW, lineSpacing, run.text, "", 0, "L", false, 0, "")
		}
		r.pdf.Ln(lineSpacing)
		r.pdf.SetFont(fontFamily, "", fontSize)
	}

	r.pdf.Ln(0.05)
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderLyric(el parser.Lyric) {
	r.pdf.SetFont(fontFamily, "I", fontSize)
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charLeft)
	r.renderFormattedBlock(el.Text, charLeft, pageWidth-charLeft-marginRight, false, true)
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

func (r *PDFRenderer) renderPageBreak() {
	r.pdf.AddPage()
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

// renderFormattedBlock renders text with inline formatting within a fixed-width block.
// It wraps the text into lines using greedy word-wrap, then renders each line chunk
// by chunk with proper font styles.
func (r *PDFRenderer) renderFormattedBlock(text string, startX, width float64, baseBold, baseItalic bool) {
	inlines := parser.ParseInline(text)
	chunks := mergeChunks(flattenInline(inlines, baseBold, baseItalic))

	if !hasFormatting(chunks, baseBold, baseItalic) {
		r.pdf.SetFont(fontFamily, r.styleStr(baseBold, baseItalic), fontSize)
		r.pdf.MultiCell(0, lineSpacing, r.tr(text), "", "L", false)
		return
	}

	// Translate each chunk to CP1252 single-byte so gofpdf's byte-based
	// width computation matches the rendered output.
	translated := make([]styledChunk, len(chunks))
	for i, c := range chunks {
		translated[i] = styledChunk{
			text:   strings.Replace(r.tr(c.text), "\r", "", -1),
			bold:   c.bold,
			italic: c.italic,
		}
	}

	// Wrap the translated chunk stream into lines.
	lines := r.wrapStyled(translated, width)

	// Render each line with per-chunk font styling.
	for _, line := range lines {
		r.pdf.SetX(startX)
		if line == nil {
			r.pdf.Ln(lineSpacing)
			continue
		}
		for _, seg := range line {
			r.pdf.SetFont(fontFamily, seg.style(), fontSize)
			w := r.pdf.GetStringWidth(seg.text)
			r.pdf.CellFormat(w, lineSpacing, seg.text, "", 0, "L", false, 0, "")
		}
		r.pdf.Ln(lineSpacing)
	}
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
	var curW float64
	for _, tok := range tokens {
		if tok.isNewline {
			lines = append(lines, cur)
			cur = nil
			curW = 0
			continue
		}
		sw := r.stringWidth(tok.text, tok.bold, tok.italic)
		if curW+sw > width && len(cur) > 0 {
			lines = append(lines, cur)
			cur = nil
			curW = 0
		}
		cur = append(cur, styledChunk{text: tok.text, bold: tok.bold, italic: tok.italic})
		curW += sw
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}
	return lines
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
