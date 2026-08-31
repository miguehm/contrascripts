package renderer

import (
	"bytes"
	"os"
	"path/filepath"
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
			parser.Character{Name: "JOHN"},
			parser.Dialogue{Text: "Hello there."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderCharacterWithExtension(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Character{Name: "JOHN", Extension: "V.O."},
			parser.Dialogue{Text: "Voice over line."},
		},
	}
	renderAndVerify(t, doc)
}

func TestRenderParenthetical(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Character{Name: "JOHN"},
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
			parser.Character{Name: "JOHN"},
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
			Title:      "A Very Long Title That Should Wrap or Display Correctly On The Title Page",
			Credit:     "Written by",
			Author:     "Author Name",
			DraftDate:  "01/01/2026",
			Contact:    "Studio Address\nLine 2\nLine 3",
			Source:     "Based on the novel by Someone",
			Revision:   "Draft 3",
			Custom:     map[string]string{"Draft": "First"},
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
			parser.Character{Name: "JOHN"},
			parser.Dialogue{Text: ""},
		},
	}

	renderAndVerify(t, doc)
}

func TestRenderSpecialCharacters(t *testing.T) {
	doc := &parser.Fountain{
		Elements: []parser.Element{
			parser.Action{Text: "He said \"hello\" and left."},
			parser.Character{Name: "JOSÉ"},
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
