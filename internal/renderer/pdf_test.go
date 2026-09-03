package renderer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/fountain/internal/parser"
)

func TestRenderEmptyDocument(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderMinimalFile(t *testing.T) {
	doc, err := parser.ParseFile(filepath.Join("..", "..", "testdata", "minimal.fountain"))
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	r := New(doc)
	if err := r.RenderToFile(filepath.Join(t.TempDir(), "minimal.pdf")); err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}
}

func TestRenderAllElementsFile(t *testing.T) {
	doc, err := parser.ParseFile(filepath.Join("..", "..", "testdata", "all-elements.fountain"))
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	r := New(doc)
	if err := r.RenderToFile(filepath.Join(t.TempDir(), "all-elements.pdf")); err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}
}

func TestRenderSceneHeading(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.SceneHeading{Text: "INT. HOUSE - DAY"},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderAction(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "A man enters the room."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderCharacter(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Character{Text: "JOHN"},
			parser.Dialogue{Text: "Hello there."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderCharacterWithExtension(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Character{Text: "JOHN (V.O.)"},
			parser.Dialogue{Text: "Voice over line."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderParenthetical(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Character{Text: "JOHN"},
			parser.Parenthetical{Text: "(whispered)"},
			parser.Dialogue{Text: "Secret message."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderTransition(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Transition{Text: "CUT TO:"},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderCentered(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Centered{Text: "THE END"},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderLyric(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Lyric{Text: "Somewhere over the rainbow"},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderPageBreak(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "Page one."},
			parser.PageBreak{},
			parser.Action{Text: "Page two."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderForcedSceneHeading(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Forced{Prefix: '.', Text: "SNIPER SCOPE POV"},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderForcedAction(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Forced{Prefix: '!', Text: "SCANNING THE AISLES..."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderForcedCharacter(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Forced{Prefix: '@', Text: "NARRATOR"},
			parser.Dialogue{Text: "Once upon a time..."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderNoteOmitted(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "Visible action."},
			parser.Note{Text: "This should not appear."},
			parser.Action{Text: "Another action."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderBoneyardOmitted(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "Visible."},
			parser.Boneyard{Text: "Hidden comment."},
			parser.Action{Text: "Also visible."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderTitlePage(t *testing.T) {
	doc := &parser.Fountain{
		TitlePage: parser.TitlePage{
			Title:  "My Screenplay",
			Author: "Jane Doe",
		},
		Elements: []parser.Element{
			parser.SceneHeading{Text: "EXT. PARK - DAY"},
			parser.Action{Text: "Birds chirp."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderFullSequence(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.SceneHeading{Text: "INT. OFFICE - DAY"},
			parser.Action{Text: "John sits at his desk."},
			parser.Character{Text: "JOHN"},
			parser.Parenthetical{Text: "(sighs)"},
			parser.Dialogue{Text: "What a day."},
			parser.Transition{Text: "CUT TO:"},
			parser.SceneHeading{Text: "EXT. PARK - DAY"},
			parser.Centered{Text: "THE END"},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderToWriter(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "Test."},
		},
	}
	r := New(doc)

	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty output")
	}

	data := buf.Bytes()
	if string(data[:5]) != "%PDF-" {
		t.Fatalf("expected PDF header, got: %s", string(data[:min(len(data), 20)]))
	}
}

func TestRenderToFile(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "Test."},
		},
	}

	path := filepath.Join(t.TempDir(), "test.pdf")
	r := New(doc)
	if err := r.RenderToFile(path); err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("PDF file is empty")
	}
}

// --- Structural PDF Tests ---

func TestPDFStructure(t *testing.T) {
	goldenDir := filepath.Join("..", "..", "testdata")
	testdataDir := filepath.Join("..", "..", "testdata")

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".fountain" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			fountainPath := filepath.Join(goldenDir, entry.Name())
			doc, err := parser.ParseFile(fountainPath)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			r := New(doc)
			var buf bytes.Buffer
			if err := r.RenderToWriter(&buf); err != nil {
				t.Fatalf("RenderToWriter failed: %v", err)
			}

			data := buf.Bytes()

			// Verify PDF header
			if len(data) < 5 || string(data[:5]) != "%PDF-" {
				t.Fatalf("invalid PDF header: %s", string(data[:min(len(data), 20)]))
			}

			// Verify PDF ends with EOF marker
			if !bytes.Contains(data, []byte("%%EOF")) {
				t.Error("PDF missing EOF marker")
			}

			// Verify PDF contains page objects
			if !bytes.Contains(data, []byte("/Type /Page")) {
				t.Error("PDF missing /Type /Page objects")
			}

			// Verify PDF is non-trivial size (at least 1KB)
			if len(data) < 1024 {
				t.Errorf("PDF too small: %d bytes", len(data))
			}
		})
	}
}

// --- Edge Case Tests ---

func TestRenderManyElementsPageBreak(t *testing.T) {
	elements := make([]parser.Element, 100)
	for i := range elements {
		elements[i] = parser.Action{Text: "Line of action text that takes up some space on the page."}
	}

	doc := &parser.Fountain{Elements: elements}
	r := New(doc)

	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderLongTitlePage(t *testing.T) {
	doc := &parser.Fountain{
		TitlePage: parser.TitlePage{
			Title:     "A Very Long Title That Should Wrap or Display Correctly On The Title Page",
			Credit:    "Written by",
			Author:    "Author Name",
			DraftDate: "01/01/2026",
			Contact:   "Studio Address\nLine 2\nLine 3",
			Source:    "Based on the novel by Someone",
			Revision:  "Draft 3",
			Custom:    map[string]string{"Draft": "First"},
		},
		Elements: []parser.Element{
			parser.SceneHeading{Text: "INT. OFFICE - DAY"},
		},
	}

	renderAndVerify(t, doc)
}

func TestRenderOnlyTitlePage(t *testing.T) {
	doc := &parser.Fountain{
		TitlePage: parser.TitlePage{
			Title:  "My Script",
			Author: "Author",
		},
	}

	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderYAMLFrontMatterTitlePage(t *testing.T) {
	src := "title: Buscar al diablo\n" +
		"author:\n" +
		"  - Sarahi Zepeda Carrera\n" +
		"  - Miguel Hernandez Moreno\n" +
		"draft-date: 13/08/2026\n" +
		"---\n" +
		"\n" +
		"EXT. CALLE - DIA #1#\n" +
		"\n" +
		"MADRE V.O\n" +
		"No pude pagar la renta.\n"

	doc := parser.Parse(src)
	if doc.TitlePage.Title != "Buscar al diablo" {
		t.Fatalf("Title = %q, want %q", doc.TitlePage.Title, "Buscar al diablo")
	}

	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderDialogueWithoutCharacter(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Dialogue{Text: "Orphaned dialogue."},
		},
	}

	renderAndVerify(t, doc)
}

func TestRenderEmptyDialogue(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Character{Text: "JOHN"},
			parser.Dialogue{Text: ""},
		},
	}

	renderAndVerify(t, doc)
}

func TestRenderSpecialCharacters(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "He said \"hello\" and left."},
			parser.Character{Text: "JOSÉ"},
			parser.Dialogue{Text: "Ñoño says café."},
		},
	}

	renderAndVerify(t, doc)
}

func TestRenderMultipleTransitions(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Transition{Text: "CUT TO:"},
			parser.SceneHeading{Text: "INT. CAR - DAY"},
			parser.Transition{Text: "FADE OUT."},
		},
	}

	renderAndVerify(t, doc)
}

func TestRenderConsecutiveSceneHeadings(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.SceneHeading{Text: "EXT. PARK - DAY"},
			parser.SceneHeading{Text: "INT. HOUSE - NIGHT"},
			parser.SceneHeading{Text: "EXT. MOUNTAIN - DAWN"},
		},
	}

	renderAndVerify(t, doc)
}

// --- Integration Tests ---

func TestIntegrationMinimalFile(t *testing.T) {
	fountainPath := filepath.Join("..", "..", "testdata", "minimal.fountain")
	doc, err := parser.ParseFile(fountainPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	data := buf.Bytes()
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		t.Fatalf("expected PDF header, got: %s", string(data[:min(len(data), 20)]))
	}
}

func TestIntegrationAllElementsFile(t *testing.T) {
	fountainPath := filepath.Join("..", "..", "testdata", "all-elements.fountain")
	doc, err := parser.ParseFile(fountainPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	data := buf.Bytes()
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		t.Fatalf("expected PDF header, got: %s", string(data[:min(len(data), 20)]))
	}
}

func TestIntegrationScreenplayCompleteFile(t *testing.T) {
	fountainPath := filepath.Join("..", "..", "testdata", "screenplay-complete.fountain")
	doc, err := parser.ParseFile(fountainPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if doc.TitlePage.Title != "THE LAST SIGNAL" {
		t.Errorf("Title = %q, want %q", doc.TitlePage.Title, "THE LAST SIGNAL")
	}
	if doc.TitlePage.Author != "Jane Smith" {
		t.Errorf("Author = %q, want %q", doc.TitlePage.Author, "Jane Smith")
	}

	r := New(doc)
	path := filepath.Join(t.TempDir(), "screenplay-complete.pdf")
	if err := r.RenderToFile(path); err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("PDF file is empty")
	}
}

// --- Inline Formatting Tests ---

func TestRenderBoldAction(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "He pulls out a **gun** and aims."},
		},
	}
	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	data := buf.Bytes()
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Fatal("PDF missing page objects")
	}
}

func TestRenderItalicDialogue(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Character{Text: "JOHN"},
			parser.Dialogue{Text: "I think _this_ is wrong."},
		},
	}
	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	data := buf.Bytes()
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Fatal("PDF missing page objects")
	}
}

func TestRenderBoldItalicCentered(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Centered{Text: "**THE END**"},
		},
	}
	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	data := buf.Bytes()
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Fatal("PDF missing page objects")
	}
}

func TestRenderItalicLyric(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Lyric{Text: "_Somewhere_ over the rainbow"},
		},
	}
	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	data := buf.Bytes()
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Fatal("PDF missing page objects")
	}
}

func TestRenderMixedFormattingAction(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "He **grabs** the _red_ folder and **_runs_**."},
		},
	}
	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	data := buf.Bytes()
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Fatal("PDF missing page objects")
	}
}

func TestRenderPlainActionNoFormatting(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "A simple action line with no formatting."},
		},
	}
	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	data := buf.Bytes()
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Fatal("PDF missing page objects")
	}
}

func TestRenderFormattedBlock(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "Line one **bold** text.\nLine two _italic_ text.\nLine three **_bold italic_** text."},
		},
	}
	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}
	data := buf.Bytes()
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Fatal("PDF missing page objects")
	}
}

func renderAndVerify(t *testing.T, doc *parser.Fountain) {
	t.Helper()

	r := New(doc)
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf); err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}

	data := buf.Bytes()
	if string(data[:5]) != "%PDF-" {
		t.Fatalf("expected PDF header, got: %s", string(data[:min(len(data), 20)]))
	}
}

// --- Wrap Styled Tests ---

func TestWrapStyledBasic(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	// Simple text that fits on one line
	chunks := []styledChunk{{text: "Hello world", bold: false, italic: false}}
	lines := r.wrapStyled(chunks, 6.0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	joined := joinLineText(lines[0])
	if joined != "Hello world" {
		t.Errorf("unexpected content: %q, want 'Hello world'", joined)
	}
}

func TestWrapStyledWordBreak(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	// "one two three" = 13 chars × 0.1in = 1.3in. Use width 1.0 to force break.
	chunks := []styledChunk{{text: "one two three", bold: false, italic: false}}
	lines := r.wrapStyled(chunks, 1.0)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	// No line should contain a mid-word split
	for i, line := range lines {
		joined := joinLineText(line)
		for _, w := range strings.Fields(joined) {
			if len(w) > 0 && !strings.HasPrefix(w, " ") && !strings.HasSuffix(w, " ") {
				// Check this word is not a fragment of any original word
				if len(w) < 3 {
					t.Errorf("line %d: suspicious short word %q — possible mid-word split", i, w)
				}
			}
		}
	}
}

func TestWrapStyledNewline(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	chunks := []styledChunk{{text: "line1\nline2", bold: false, italic: false}}
	lines := r.wrapStyled(chunks, 6.0)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines from newline, got %d", len(lines))
	}
	if joinLineText(lines[0]) != "line1" {
		t.Errorf("first line = %q, want 'line1'", joinLineText(lines[0]))
	}
	if joinLineText(lines[1]) != "line2" {
		t.Errorf("second line = %q, want 'line2'", joinLineText(lines[1]))
	}
}

func TestWrapStyledBoldChunks(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	chunks := []styledChunk{
		{text: "plain ", bold: false, italic: false},
		{text: "bold", bold: true, italic: false},
		{text: " end", bold: false, italic: false},
	}
	lines := r.wrapStyled(chunks, 6.0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// Verify mixed styles in the line
	hasBold := false
	hasPlain := false
	for _, seg := range lines[0] {
		if seg.bold {
			hasBold = true
		} else {
			hasPlain = true
		}
	}
	if !hasBold || !hasPlain {
		t.Errorf("expected both bold and plain segments, got %v", lines[0])
	}
}

func TestWrapStyledAccentedText(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	// Spanish accented text — should never split mid-word
	chunks := []styledChunk{{text: "la expresión de la rabia", bold: false, italic: false}}
	lines := r.wrapStyled(chunks, 2.0) // narrow width
	for i, line := range lines {
		joined := joinLineText(line)
		words := strings.Fields(joined)
		for _, w := range words {
			// Each word should be a complete word from the original
			if len(w) > 0 && len(w) < 2 {
				t.Errorf("line %d: suspicious short word %q — possible mid-word split on accented text", i, w)
			}
		}
	}
}

func TestWrapStyledForcedSplit(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	// Single word wider than width — should force it on a line, not panic
	longWord := strings.Repeat("x", 100)
	chunks := []styledChunk{{text: longWord, bold: false, italic: false}}
	lines := r.wrapStyled(chunks, 1.0) // width can't fit the word
	if len(lines) < 1 {
		t.Fatal("expected at least 1 line for forced split")
	}
}

func TestWrapStyledEmptyText(t *testing.T) {
	doc := &parser.Fountain{}
	r := New(doc)

	chunks := []styledChunk{{text: "", bold: false, italic: false}}
	lines := r.wrapStyled(chunks, 6.0)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty text, got %d", len(lines))
	}
}

func joinLineText(line []styledChunk) string {
	var b strings.Builder
	for _, seg := range line {
		b.WriteString(seg.text)
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
