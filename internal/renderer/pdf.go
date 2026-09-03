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

	lineHeight = 6.0 // points per line at 12pt Courier
	fontSize   = 12.0
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
	pdf.SetFont("Courier", "", fontSize)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

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
			r.pdf.SetFont("Courier", "", 10)
			r.pdf.SetXY(pageWidth-marginRight-0.5, 0.3)
			r.pdf.CellFormat(0.5, 0.3, fmt.Sprintf("%d", r.pdf.PageNo()-1), "", 0, "R", false, 0, "")
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
		tp.Revision != "" || len(tp.Custom) > 0
}

func (r *PDFRenderer) renderTitlePage() {
	r.pdf.AddPage()
	r.pdf.SetFont("Courier", "", fontSize)

	tp := r.doc.TitlePage
	y := 3.5

	if tp.Title != "" {
		r.pdf.SetFont("Courier", "B", 24)
		r.pdf.SetXY(0, y)
		r.pdf.CellFormat(pageWidth, 12, r.tr(strings.ToUpper(tp.Title)), "", 1, "C", false, 0, "")
		y += 0.6
	}

	r.pdf.SetFont("Courier", "", fontSize)

	fields := []struct {
		label string
		value string
	}{
		{"Credit", tp.Credit},
		{"Author", tp.Author},
		{"Draft Date", tp.DraftDate},
		{"Contact", tp.Contact},
		{"Source", tp.Source},
		{"Revision", tp.Revision},
	}

	for _, f := range fields {
		if f.value != "" {
			r.pdf.SetXY(0, y)
			r.pdf.CellFormat(pageWidth, 0.3, r.tr(f.value), "", 1, "C", false, 0, "")
			y += 0.35
		}
	}

	for k, v := range tp.Custom {
		if v != "" {
			r.pdf.SetXY(0, y)
			r.pdf.CellFormat(pageWidth, 0.3, r.tr(fmt.Sprintf("%s: %s", k, v)), "", 1, "C", false, 0, "")
			y += 0.35
		}
	}

	r.pdf.AddPage()
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
	r.pdf.SetFont("Courier", "", fontSize)
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charLeft)
	r.pdf.MultiCell(0, 0.18, r.tr(strings.ToUpper(el.Text)), "", "L", false)
	r.pdf.Ln(0.15)
	r.pdf.SetFont("Courier", "", fontSize)
}

func (r *PDFRenderer) renderAction(el parser.Action) {
	r.pdf.Ln(0.05)
	r.pdf.SetFont("Courier", "", fontSize)
	r.pdf.SetLeftMargin(actionLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(actionLeft)
	r.renderFormattedBlock(el.Text, actionLeft, pageWidth-actionLeft-marginRight, false, false)
	r.pdf.Ln(0.05)
}

func (r *PDFRenderer) renderCharacter(el parser.Character) {
	r.pdf.Ln(0.1)
	r.pdf.SetFont("Courier", "", fontSize)
	r.pdf.SetLeftMargin(charNameLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charNameLeft)

	name := strings.ToUpper(el.Name)
	if el.Extension != "" {
		name += " " + el.Extension
	}

	r.pdf.MultiCell(0, 0.18, r.tr(name), "", "L", false)
	r.pdf.SetFont("Courier", "", fontSize)
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
	r.pdf.SetFont("Courier", "", fontSize)
	text := r.tr(strings.ToUpper(el.Text))
	w := r.pdf.GetStringWidth(text)
	x := pageWidth - marginRight - w
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	r.pdf.SetX(x)
	r.pdf.CellFormat(w, 0.18, text, "", 1, "L", false, 0, "")
	r.pdf.Ln(0.1)
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderCentered(el parser.Centered) {
	r.pdf.Ln(0.05)
	r.pdf.SetFont("Courier", "", fontSize)
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	r.pdf.SetX(0)

	inlines := parser.ParseInline(el.Text)
	chunks := mergeChunks(flattenInline(inlines, false, false))

	if !hasFormatting(chunks, false, false) {
		r.pdf.CellFormat(pageWidth, 0.18, r.tr(el.Text), "", 1, "C", false, 0, "")
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
			r.pdf.SetFont("Courier", run.style, fontSize)
			runW := r.pdf.GetStringWidth(run.text)
			r.pdf.CellFormat(runW, 0.18, run.text, "", 0, "L", false, 0, "")
		}
		r.pdf.Ln(0.18)
		r.pdf.SetFont("Courier", "", fontSize)
	}

	r.pdf.Ln(0.05)
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderLyric(el parser.Lyric) {
	r.pdf.SetFont("Courier", "I", fontSize)
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charLeft)
	r.renderFormattedBlock(el.Text, charLeft, pageWidth-charLeft-marginRight, false, true)
	r.pdf.SetFont("Courier", "", fontSize)
}

func (r *PDFRenderer) renderForced(el parser.Forced) {
	switch el.Prefix {
	case '.':
		r.renderSceneHeading(parser.SceneHeading{Text: el.Text})
	case '!':
		r.renderAction(parser.Action{Text: el.Text})
	case '@':
		r.renderCharacter(parser.Character{Name: el.Text})
	default:
		r.renderAction(parser.Action{Text: el.Text})
	}
}

func (r *PDFRenderer) renderPageBreak() {
	r.pdf.AddPage()
}

// styledChunk represents a run of text with a specific font style.
type styledChunk struct {
	text  string
	bold  bool
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
		r.pdf.SetFont("Courier", r.styleStr(baseBold, baseItalic), fontSize)
		r.pdf.MultiCell(0, 0.18, r.tr(text), "", "L", false)
		return
	}

	// Translate each chunk to CP1252 single-byte so gofpdf's byte-based
	// width computation matches the rendered output.
	translated := make([]styledChunk, len(chunks))
	for i, c := range chunks {
		translated[i] = styledChunk{
			text:  strings.Replace(r.tr(c.text), "\r", "", -1),
			bold:  c.bold,
			italic: c.italic,
		}
	}

	// Wrap the translated chunk stream into lines.
	lines := r.wrapStyled(translated, width)

	// Render each line with per-chunk font styling.
	for _, line := range lines {
		r.pdf.SetX(startX)
		if line == nil {
			r.pdf.Ln(0.18)
			continue
		}
		for _, seg := range line {
			r.pdf.SetFont("Courier", seg.style(), fontSize)
			w := r.pdf.GetStringWidth(seg.text)
			r.pdf.CellFormat(w, 0.18, seg.text, "", 0, "L", false, 0, "")
		}
		r.pdf.Ln(0.18)
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
	r.pdf.SetFont("Courier", styledChunk{bold: bold, italic: italic}.style(), fontSize)
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
