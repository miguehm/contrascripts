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

	charLeft    = 1.5
	charRight   = 7.5
	actionLeft  = 1.5
	actionRight = 7.5
	charNameLeft = 3.7
	parenLeft   = 3.1
	parenRight  = 5.0
	dialogLeft  = 2.5
	dialogRight = 6.0
	transRight  = 1.0

	lineHeight = 6.0 // points per line at 12pt Courier
	fontSize   = 12.0
)

type PDFRenderer struct {
	pdf *gofpdf.Fpdf
	doc *parser.Fountain
}

func New(doc *parser.Fountain) *PDFRenderer {
	pdf := gofpdf.New("P", "in", "Letter", "")
	pdf.SetMargins(marginLeft, marginTop, marginRight)
	pdf.SetAutoPageBreak(true, marginBottom)
	pdf.SetFont("Courier", "", fontSize)

	return &PDFRenderer{
		pdf: pdf,
		doc: doc,
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
	r.pdf.SetHeaderFunc(func() {
		if r.pdf.PageNo() > 1 {
			r.pdf.SetFont("Courier", "", 10)
			r.pdf.SetXY(pageWidth-marginRight-0.5, 0.3)
			r.pdf.CellFormat(0.5, 0.3, fmt.Sprintf("%d", r.pdf.PageNo()-1), "", 0, "R", false, 0, "")
		}
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
		r.pdf.CellFormat(pageWidth, 12, strings.ToUpper(tp.Title), "", 1, "C", false, 0, "")
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
			r.pdf.CellFormat(pageWidth, 0.3, f.value, "", 1, "C", false, 0, "")
			y += 0.35
		}
	}

	for k, v := range tp.Custom {
		if v != "" {
			r.pdf.SetXY(0, y)
			r.pdf.CellFormat(pageWidth, 0.3, fmt.Sprintf("%s: %s", k, v), "", 1, "C", false, 0, "")
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
	r.pdf.SetFont("Courier", "B", fontSize)
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charLeft)
	r.pdf.MultiCell(0, 0.18, strings.ToUpper(el.Text), "", "L", false)
	r.pdf.Ln(0.15)
	r.pdf.SetFont("Courier", "", fontSize)
}

func (r *PDFRenderer) renderAction(el parser.Action) {
	r.pdf.Ln(0.05)
	r.pdf.SetFont("Courier", "", fontSize)
	r.pdf.SetLeftMargin(actionLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(actionLeft)
	r.pdf.MultiCell(0, 0.18, el.Text, "", "L", false)
	r.pdf.Ln(0.05)
}

func (r *PDFRenderer) renderCharacter(el parser.Character) {
	r.pdf.Ln(0.1)
	r.pdf.SetFont("Courier", "B", fontSize)
	r.pdf.SetLeftMargin(charNameLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charNameLeft)

	name := strings.ToUpper(el.Name)
	if el.Extension != "" {
		name += " " + el.Extension
	}

	r.pdf.MultiCell(0, 0.18, name, "", "L", false)
	r.pdf.SetFont("Courier", "", fontSize)
}

func (r *PDFRenderer) renderDialogue(el parser.Dialogue) {
	r.pdf.SetLeftMargin(dialogLeft)
	r.pdf.SetRightMargin(pageWidth - dialogRight)
	r.pdf.SetX(dialogLeft)
	r.pdf.MultiCell(0, 0.18, el.Text, "", "L", false)
	r.pdf.Ln(0.05)
}

func (r *PDFRenderer) renderParenthetical(el parser.Parenthetical) {
	r.pdf.SetLeftMargin(parenLeft)
	r.pdf.SetRightMargin(pageWidth - parenRight)
	r.pdf.SetX(parenLeft)
	r.pdf.MultiCell(0, 0.18, el.Text, "", "L", false)
}

func (r *PDFRenderer) renderTransition(el parser.Transition) {
	r.pdf.Ln(0.1)
	r.pdf.SetFont("Courier", "", fontSize)
	text := strings.ToUpper(el.Text)
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
	r.pdf.CellFormat(pageWidth, 0.18, el.Text, "", 1, "C", false, 0, "")
	r.pdf.Ln(0.05)
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
}

func (r *PDFRenderer) renderLyric(el parser.Lyric) {
	r.pdf.SetFont("Courier", "I", fontSize)
	r.pdf.SetLeftMargin(charLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetX(charLeft)
	r.pdf.MultiCell(0, 0.18, el.Text, "", "L", false)
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
