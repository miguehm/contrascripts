package renderer

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/user/fountain/internal/parser"
)

// The constants below restate the renderer geometry in PDF points so the tests
// assert the published screenplay spec rather than whatever the renderer
// happens to do today.
const (
	ptPageHeight = 11.0 * 72
	ptMarginTop  = 1.0 * 72
	ptLinePitch  = 12.0 // 12pt Courier single spaced, i.e. six lines per inch
	ptBlankLine  = 2 * ptLinePitch
	// Two blank lines, which is what the format puts above a scene heading and
	// after a transition. The gap between two baselines is always one more
	// pitch than the number of blank lines between them.
	ptTwoBlankLines = 3 * ptLinePitch
	// gofpdf puts a baseline at .5*h + .3*fontSize below the top of its box.
	ptBaselineOff = 0.5*ptLinePitch + 0.3*12.0
)

// textOp matches the positioning operator gofpdf emits for every left-aligned
// or centered cell. Compression is disabled in baselineRuns so these appear
// verbatim in the content stream.
var textOp = regexp.MustCompile(`BT (\d+\.\d\d) (\d+\.\d\d) Td \((.*?)\)Tj ET`)

type textRun struct {
	x, y float64
	text string
}

// textRuns renders src and returns every positioning operator gofpdf emitted, in
// content-stream order and ungrouped. The margin marks need this level: two
// separate cells in the header band legitimately share a baseline, so grouping
// them into one visual line would weld the CONTINUED: head onto the page number.
func textRuns(t *testing.T, src string) []textRun {
	t.Helper()

	r := New(parser.Parse(src))
	// gofpdf compresses page streams by default; the assertions below need the
	// positioning operators in the clear.
	r.pdf.SetCompression(false)

	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter() error = %v, want nil", err)
	}

	matches := textOp.FindAllStringSubmatch(buf.String(), -1)
	runs := make([]textRun, 0, len(matches))
	for _, m := range matches {
		runs = append(runs, textRun{
			x:    mustFloat(t, m[1]),
			y:    mustFloat(t, m[2]),
			text: decodePDFString(m[3]),
		})
	}
	return runs
}

// baselineLines renders src and groups the operators into the visual lines they
// form, in page order. Everything sharing a baseline is one line, and a baseline
// higher than the previous one means a new page began.
//
// Measuring the result this way is what lets the tests check layout against the
// published format rather than against the renderer's drawing strategy.
func baselineLines(t *testing.T, src string) []textRun {
	t.Helper()

	var lines []textRun
	prevY := 0.0
	for _, run := range textRuns(t, src) {
		n := len(lines)
		if n == 0 || run.y > prevY+0.5 || math.Abs(run.y-lines[n-1].y) > 0.01 {
			lines = append(lines, textRun{y: run.y})
		}
		line := &lines[len(lines)-1]
		if line.text == "" {
			line.x = run.x
		} else if run.x < line.x {
			line.x = run.x
		}
		line.text += run.text
		prevY = run.y
	}
	return lines
}

// decodePDFString undoes the two transformations between the Go source string
// and the PDF literal string: parentheses and backslashes are escaped byte by
// byte, and the embedded font makes gofpdf emit UTF-16BE.
//
// The escapes have to be undone first and on their own, because a parenthesis
// inside a two-byte character escapes the low byte of a pair, and reversing
// that out of order would desynchronise every pair after it. Stripping the NUL
// bytes instead, as an ASCII-only script never notices, would leave accented
// characters as lone Latin-1 bytes.
func decodePDFString(s string) string {
	raw := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			raw = append(raw, s[i])
			continue
		}
		raw = append(raw, s[i])
	}
	if !looksUTF16BE(raw) {
		// The core font fallback is single byte, so there is nothing to decode.
		return string(raw)
	}
	var b strings.Builder
	for i := 0; i+1 < len(raw); i += 2 {
		b.WriteRune(rune(raw[i])<<8 | rune(raw[i+1]))
	}
	return b.String()
}

// looksUTF16BE reports whether raw is a UTF-16BE stream, which shows up as an
// even length whose leading byte of every pair is zero.
func looksUTF16BE(raw []byte) bool {
	if len(raw) == 0 || len(raw)%2 != 0 {
		return false
	}
	for i := 0; i < len(raw); i += 2 {
		if raw[i] != 0 {
			return false
		}
	}
	return true
}

// mustFloat parses a coordinate gofpdf emitted, failing the test if it is not
// a number.
func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("ParseFloat(%q) error = %v, want nil", s, err)
	}
	return v
}

func textsOf(runs []textRun) []string {
	out := make([]string, len(runs))
	for i, r := range runs {
		out[i] = r.text
	}
	return out
}

func assertWithin(t *testing.T, what string, got, want, tolerance float64) {
	t.Helper()
	if diff := got - want; diff > tolerance || diff < -tolerance {
		t.Errorf("%s = %.4fpt, want %.4fpt", what, got, want)
	}
}

// TestBodyStartsOnTheTopMargin pins the first baseline of the screenplay body
// to exactly the 1" top margin, with no leading gap before the first element.
func TestBodyStartsOnTheTopMargin(t *testing.T) {
	runs := baselineLines(t, "INT. HOUSE - DAY\n\nA door stands open.\n")
	if len(runs) == 0 {
		t.Fatal("baselineRuns() = 0 runs, want at least 1")
	}

	assertWithin(t, "first baseline y", runs[0].y, ptPageHeight-ptMarginTop-ptBaselineOff, 0.01)
}

// TestLineGridIsSingleSpaced checks that a block too wide for its column wraps
// at exactly one 12pt line, which is the industry rule of six lines per
// vertical inch. The action is one long source line so that the breaks under
// test are the renderer's own soft wraps.
func TestLineGridIsSingleSpaced(t *testing.T) {
	action := strings.Repeat("the corridor goes on and on and on. ", 5)
	src := "INT. HOUSE - DAY\n\n" + action + "\n"

	runs := baselineLines(t, src)
	if len(runs) < 4 {
		t.Fatalf("baselineRuns(%q) = %d runs %v, want at least 4 (heading + 3 wrapped lines)", src, len(runs), textsOf(runs))
	}

	// Run 0 is the scene heading; everything after it is the wrapped action.
	for i := 2; i < len(runs); i++ {
		assertWithin(t, "wrapped line pitch", runs[i-1].y-runs[i].y, ptLinePitch, 0.01)
	}
}

// TestBlockGaps covers the distance from the last baseline of one block to the
// first baseline of the next. The standard separates every element from its
// neighbour by exactly one blank line, except inside a character, parenthetical
// and dialogue group, which stays single spaced.
func TestBlockGaps(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantRuns []string
		wantGaps []float64
	}{
		{
			name: "action to action is one blank line",
			src:  "INT. HOUSE - DAY\n\nRain hammers the window.\n\nA door swings open.\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "Rain hammers the window.", "A door swings open.",
			},
			wantGaps: []float64{ptBlankLine, ptBlankLine},
		},
		{
			name: "action to scene heading is two blank lines",
			src:  "INT. HOUSE - DAY\n\nRain hammers the window.\n\nEXT. STREET - NIGHT\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "Rain hammers the window.", "EXT. STREET - NIGHT",
			},
			wantGaps: []float64{ptBlankLine, ptTwoBlankLines},
		},
		{
			name: "scene heading to scene heading is two blank lines",
			src:  "INT. HOUSE - DAY\n\nEXT. STREET - NIGHT\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "EXT. STREET - NIGHT",
			},
			wantGaps: []float64{ptTwoBlankLines},
		},
		{
			name: "transition is followed by two blank lines",
			src:  "Rain.\n\nMARA\nIt is not stopping.\n\nCUT TO:\n\nA door swings open.\n",
			wantRuns: []string{
				"Rain.", "MARA", "It is not stopping.", "CUT TO:", "A door swings open.",
			},
			wantGaps: []float64{ptBlankLine, ptLinePitch, ptBlankLine, ptTwoBlankLines},
		},
		{
			name: "dialogue group is single spaced inside and one blank line before",
			src:  "INT. HOUSE - DAY\n\nRain hammers the window.\n\nMARA\n(whispering)\nIt is not stopping.\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "Rain hammers the window.",
				"MARA", "(whispering)", "It is not stopping.",
			},
			wantGaps: []float64{ptBlankLine, ptBlankLine, ptLinePitch, ptLinePitch},
		},
		{
			name: "dialogue to transition is one blank line",
			src:  "INT. HOUSE - DAY\n\nRain hammers the window.\n\nMARA\nIt is not stopping.\n\nCUT TO:\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "Rain hammers the window.",
				"MARA", "It is not stopping.", "CUT TO:",
			},
			wantGaps: []float64{ptBlankLine, ptBlankLine, ptLinePitch, ptBlankLine},
		},
		{
			name: "lyric is one blank line from the surrounding action",
			src:  "INT. HOUSE - DAY\n\nRain hammers the window.\n\n~ And the rain falls down\n\nA door swings open.\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "Rain hammers the window.",
				"And the rain falls down", "A door swings open.",
			},
			wantGaps: []float64{ptBlankLine, ptBlankLine, ptBlankLine},
		},
		{
			// The regression this file exists to catch: a multi-line paragraph
			// is one element, so its soft line breaks are single spaced rather
			// than separated by a blank line.
			name: "a multi-line action paragraph is single spaced internally",
			src:  "INT. HOUSE - DAY\n\nRain hammers the window.\nThe glass holds.\n\nA door swings open.\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "Rain hammers the window.", "The glass holds.", "A door swings open.",
			},
			wantGaps: []float64{ptBlankLine, ptLinePitch, ptBlankLine},
		},
		{
			name: "a multi-line speech is single spaced internally",
			src:  "INT. HOUSE - DAY\n\nMARA\nIt is not stopping.\nNot tonight.\n\nA door swings open.\n",
			wantRuns: []string{
				"INT. HOUSE - DAY", "MARA", "It is not stopping.", "Not tonight.", "A door swings open.",
			},
			wantGaps: []float64{ptBlankLine, ptLinePitch, ptLinePitch, ptBlankLine},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runs := baselineLines(t, tt.src)
			if len(runs) != len(tt.wantRuns) {
				t.Fatalf("baselineRuns(%q) = %d runs %v, want %d", tt.src, len(runs), textsOf(runs), len(tt.wantRuns))
			}
			for i, want := range tt.wantRuns {
				if got := runs[i].text; got != want {
					t.Errorf("run %d text = %q, want %q", i, got, want)
				}
			}
			for i, want := range tt.wantGaps {
				got := runs[i].y - runs[i+1].y
				assertWithin(t, "baseline gap before "+tt.wantRuns[i+1], got, want, 0.01)
			}
		})
	}
}

// TestInvisibleElementsDoNotConsumeSpacing checks that a note or boneyard, which
// is parsed but never printed, does not eat the blank line that separates the
// two visible elements around it.
func TestInvisibleElementsDoNotConsumeSpacing(t *testing.T) {
	runs := baselineLines(t, "INT. HOUSE - DAY\n\nRain hammers the window.\n\n[[a note]]\n\nA door swings open.\n")
	want := []string{"INT. HOUSE - DAY", "Rain hammers the window.", "A door swings open."}
	if len(runs) != len(want) {
		t.Fatalf("baselineRuns() = %d runs %v, want %d", len(runs), textsOf(runs), len(want))
	}
	for i, text := range want {
		if runs[i].text != text {
			t.Errorf("run %d text = %q, want %q", i, runs[i].text, text)
		}
	}
	assertWithin(t, "baseline gap across a note", runs[1].y-runs[2].y, ptBlankLine, 0.01)
}

// TestColumnPositions checks that each element starts at its specified column.
// The columns are the reason the 1mm gofpdf cell margin has to be disabled: it
// would shift every one of them by 2.835pt.
func TestColumnPositions(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		runIdx  int
		wantIn  float64
		wantTie string
	}{
		{name: "scene heading", src: "INT. HOUSE - DAY\n\nRain.\n", runIdx: 0, wantIn: 1.5},
		{name: "action", src: "INT. HOUSE - DAY\n\nRain hammers the window.\n", runIdx: 1, wantIn: 1.5},
		{name: "character cue", src: "Rain.\n\nMARA\nIt is not stopping.\n", runIdx: 1, wantIn: 3.7},
		{name: "parenthetical", src: "Rain.\n\nMARA\n(whispering)\nIt is not stopping.\n", runIdx: 2, wantIn: 3.1},
		{name: "dialogue", src: "Rain.\n\nMARA\nIt is not stopping.\n", runIdx: 2, wantIn: 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runs := baselineLines(t, tt.src)
			if len(runs) <= tt.runIdx {
				t.Fatalf("baselineRuns(%q) = %d runs %v, want more than %d", tt.src, len(runs), textsOf(runs), tt.runIdx)
			}
			assertWithin(t, "left column", runs[tt.runIdx].x/72, tt.wantIn, 0.001)
		})
	}
}

// TestTransitionsAreFlushRight checks that a transition ends exactly on the 1"
// right margin. The 1mm cell margin gofpdf applies by default would leave it
// 2.835pt short.
func TestTransitionsAreFlushRight(t *testing.T) {
	runs := baselineLines(t, "Rain.\n\nMARA\nIt is not stopping.\n\nCUT TO:\n")
	if len(runs) == 0 {
		t.Fatal("baselineRuns() = 0 runs, want at least 1")
	}

	last := runs[len(runs)-1]
	if last.text != "CUT TO:" {
		t.Fatalf("last run text = %q, want %q", last.text, "CUT TO:")
	}

	// Courier Prime advances 0.5996em, so seven characters of "CUT TO:" are
	// 50.37pt rather than a flat 50.4. The slack is why this is a tolerance
	// check and not an equality check.
	const cutToWidth = 7 * 7.1953125
	assertWithin(t, "transition right edge", last.x+cutToWidth, 7.5*72, 0.5)
}

// TestPageBreakResumesOnTheTopMargin checks that an explicit page break puts
// the next element back on the top margin, with no leftover gap from the block
// that preceded the break.
func TestPageBreakResumesOnTheTopMargin(t *testing.T) {
	runs := baselineLines(t, "INT. HOUSE - DAY\n\nRain hammers the window.\n\n=== \n\nEXT. STREET - NIGHT\n")
	if len(runs) == 0 {
		t.Fatal("baselineRuns() = 0 runs, want at least 1")
	}

	want := ptPageHeight - ptMarginTop - ptBaselineOff
	for _, r := range runs {
		if r.text == "EXT. STREET - NIGHT" {
			assertWithin(t, "first baseline after a page break", r.y, want, 0.01)
			return
		}
	}
	t.Fatalf("baselineRuns() = %v, want a run with text %q", textsOf(runs), "EXT. STREET - NIGHT")
}

// TestActionLineHoldsSixtyCharacters pins the 6.0in text column to the sixty
// Courier characters it is defined to hold. gofpdf's default 1mm cell margin
// used to steal enough width to wrap a full line onto a second one.
func TestActionLineHoldsSixtyCharacters(t *testing.T) {
	tests := []struct {
		name  string
		chars int
		want  int
	}{
		{name: "sixty characters fit on one line", chars: 60, want: 1},
		{name: "sixty one characters wrap", chars: 61, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "INT. HOUSE - DAY\n\n" + strings.Repeat("x", tt.chars) + "\n"
			runs := baselineLines(t, src)
			if len(runs) != tt.want+1 {
				t.Fatalf("baselineLines(%q) = %d lines %v, want %d (heading + %d action lines)",
					src, len(runs), textsOf(runs), tt.want+1, tt.want)
			}
		})
	}
}

// TestUnbrokenAccentedWordWrapsOnRuneBoundaries checks a word too wide for the
// column is divided, and that no division lands inside a multi-byte character.
// Cutting mid-character hands gofpdf a string whose final byte is a lead byte
// with no continuation, and its UTF-16 conversion reads past the end and panics,
// so the pieces have to rejoin into the original word exactly.
func TestUnbrokenAccentedWordWrapsOnRuneBoundaries(t *testing.T) {
	// Seventy characters with no space between them, so the wrapper has to break
	// inside a single token. Every character is two bytes.
	word := strings.Repeat("á", 70)
	if !utf8.ValidString(word) {
		t.Fatal("test input is not valid UTF-8")
	}

	lines := baselineLines(t, "INT. HOUSE - DAY\n\n"+word+"\n")
	if len(lines) < 3 {
		t.Fatalf("baselineLines() = %d lines %v, want the word divided", len(lines), textsOf(lines))
	}

	var got strings.Builder
	for _, l := range lines[1:] {
		if !utf8.ValidString(l.text) {
			t.Fatalf("line %q is not valid UTF-8, so it was cut mid-character", l.text)
		}
		got.WriteString(l.text)
	}
	if got.String() != word {
		t.Errorf("the divided word rejoined as %q, want %q", got.String(), word)
	}
}

// TestLongestPrefixStopsOnRuneBoundaries checks the width helper directly, since
// every forced split depends on it returning a byte count on a character edge.
func TestLongestPrefixStopsOnRuneBoundaries(t *testing.T) {
	const twoByte = "áéíóúñ" // six runes, twelve bytes
	r := New(&parser.Fountain{})
	oneRune := r.stringWidth("á", false, false)

	tests := []struct {
		name  string
		text  string
		width float64
		want  int
	}{
		{name: "the whole string fits", text: twoByte, width: oneRune * 6, want: 12},
		{name: "two runes fit", text: twoByte, width: oneRune * 2, want: 4},
		{name: "one rune fits", text: twoByte, width: oneRune * 1, want: 2},
		{name: "not even one rune fits", text: twoByte, width: oneRune * 0.5, want: 0},
		{name: "an empty string", text: "", width: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := longestPrefix(r, tt.text, tt.width, false, false)
			if got != tt.want {
				t.Fatalf("longestPrefix(%q, %v) = %d, want %d", tt.text, tt.width, got, tt.want)
			}
			if !utf8.ValidString(tt.text[:got]) {
				t.Errorf("longestPrefix(%q, %v) cut inside a character", tt.text, tt.width)
			}
		})
	}
}
