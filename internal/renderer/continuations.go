package renderer

import (
	"fmt"
	"strings"

	"github.com/user/fountain/internal/parser"
)

// Continuation notation belongs to a production script. A spec script is read
// by people who turn the page, and the format guides say the notation only
// clutters it, so it is emitted only when the title page asks for it with
// `production: true`.
const (
	moreText    = "(MORE)"
	contdText   = "(CONT'D)"
	continuedTx = "(CONTINUED)"
	continuedHd = "CONTINUED:"
	// continuFontSize is smaller than the body, which is how the production
	// format sets these apart from the script itself.
	continuFontSize = 10.0
	// continuBand is the height of the cell a margin mark is drawn in, and
	// continuEdge the distance from the paper's edge to the top of that cell.
	// Both marks use the page number's 0.5in band so they line up with it; the
	// foot mirrors the band at the bottom, which is 1in of clear margin.
	continuBand = 0.25
	continuEdge = 0.5
)

// production reports whether the document asked for the production notation.
func (r *PDFRenderer) production() bool { return r.doc.TitlePage.Production }

// drawContinuedCue reprints a cue at the top of the page a speech continues
// onto, with its (CONT'D) extension.
func (r *PDFRenderer) drawContinuedCue(cue parser.Character) {
	r.pdf.SetLeftMargin(charNameLeft)
	r.pdf.SetRightMargin(marginRight)
	lines := r.measureBlock(continuedCue(cue).Text, pageWidth-charNameLeft-marginRight, false, false)
	r.drawVisualLines(charNameLeft, pageWidth-charNameLeft-marginRight, lines, 0, len(lines), false)
	r.pdf.SetLeftMargin(dialogLeft)
	r.pdf.SetRightMargin(pageWidth - dialogRight)
}

// drawMore writes (MORE) under a speech that continues over the page break, at
// the character cue indent and with no blank line between it and the dialogue.
func (r *PDFRenderer) drawMore() {
	r.pdf.SetFont(fontFamily, "", continuFontSize)
	r.pdf.SetX(charNameLeft)
	r.pdf.CellFormat(0.5, lineSpacing, r.tr(moreText), "", 0, "L", false, 0, "")
	r.blankLine()
	r.pdf.SetFont(fontFamily, "", fontSize)
}

// continuedCue is the cue as it is reprinted at the top of the next page when a
// speech continues. A name that already carries an extension such as (V.O.) or
// (O.S.) keeps it, and the (CONT'D) follows after a semicolon and a space rather
// than displacing it.
func continuedCue(el parser.Character) parser.Character {
	if hasExtension(el.Text) {
		return parser.Character{Text: el.Text + "; " + contdText}
	}
	return parser.Character{Text: el.Text + " " + contdText}
}

// hasExtension reports whether a character name already ends in an extension,
// which is a closing parenthesis.
func hasExtension(name string) bool {
	name = strings.TrimSpace(name)
	return strings.HasSuffix(name, ")")
}

// drawContinued writes (CONTINUED) at the bottom right of a page whose scene
// carries on, for a production script.
//
// The Y is set explicitly because the footer runs with the cursor wherever the
// page happened to end, which is inside the text block: the last reachable body
// baseline is 86.4pt and the mark landed at 87pt, so a full page printed the two
// on top of each other. Dropping it into the bottom margin, below the last line
// of script, is where the format puts it and where nothing can reach it.
func (r *PDFRenderer) drawContinued() {
	r.pdf.SetFont(fontFamily, "", continuFontSize)
	w := r.pdf.GetStringWidth(continuedTx)
	r.pdf.SetLeftMargin(0)
	r.pdf.SetRightMargin(0)
	r.pdf.SetXY(pageWidth-marginRight-w, pageHeight-continuEdge-continuBand)
	r.pdf.CellFormat(w, continuBand, r.tr(continuedTx), "", 0, "L", false, 0, "")
	r.pdf.SetLeftMargin(marginLeft)
	r.pdf.SetRightMargin(marginRight)
	r.pdf.SetFont(fontFamily, "", fontSize)
}

// drawContinuedHead writes CONTINUED: n in the band above the text block, where
// the page number already sits on the other side, and n is the ordinal of the
// scene the page belongs to. The mark is placed there rather than at the top of
// the text block for two reasons: nothing body-related is up there, so it cannot
// collide with the first line of the page, and it does not consume a line of the
// 54-line grid, so a continued page still holds the same amount of script.
func (r *PDFRenderer) drawContinuedHead(scene int) {
	// Thirteen characters of "CONTINUED: 99" at 10pt is 1.08in, so 1.5in of cell
	// is enough without crowding the page number, which starts at 7.0in.
	r.pdf.SetFont(fontFamily, "", continuFontSize)
	r.pdf.SetXY(actionLeft, continuEdge)
	r.pdf.CellFormat(1.5, continuBand, r.tr(fmt.Sprintf("%s %d", continuedHd, scene)), "", 0, "L", false, 0, "")
}
