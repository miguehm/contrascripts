package parser

import (
	"os"
	"testing"
)

// --- Title Page Tests ---

func TestParseTitlePage(t *testing.T) {
	input := "Title: My Script\nAuthor: Jane Doe\n\nINT. HOUSE - DAY\n"
	f := Parse(input)

	if f.TitlePage.Title != "My Script" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "My Script")
	}
	if f.TitlePage.Author != "Jane Doe" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "Jane Doe")
	}
}

func TestParseTitlePageAllKnownFields(t *testing.T) {
	input := "Title: T\nCredit: C\nAuthor: A\nDraft date: D\nContact: CO\nSource: S\nRevision: R\n\nINT. X - DAY\n"
	f := Parse(input)

	if f.TitlePage.Title != "T" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "T")
	}
	if f.TitlePage.Credit != "C" {
		t.Errorf("Credit = %q, want %q", f.TitlePage.Credit, "C")
	}
	if f.TitlePage.Author != "A" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "A")
	}
	if f.TitlePage.DraftDate != "D" {
		t.Errorf("DraftDate = %q, want %q", f.TitlePage.DraftDate, "D")
	}
	if f.TitlePage.Contact != "CO" {
		t.Errorf("Contact = %q, want %q", f.TitlePage.Contact, "CO")
	}
	if f.TitlePage.Source != "S" {
		t.Errorf("Source = %q, want %q", f.TitlePage.Source, "S")
	}
	if f.TitlePage.Revision != "R" {
		t.Errorf("Revision = %q, want %q", f.TitlePage.Revision, "R")
	}
}

func TestParseTitlePageCustomFields(t *testing.T) {
	input := "Title: T\nCustom Key: Custom Value\n\nINT. X - DAY\n"
	f := Parse(input)

	if f.TitlePage.Custom["Custom Key"] != "Custom Value" {
		t.Errorf("Custom[Custom Key] = %q, want %q", f.TitlePage.Custom["Custom Key"], "Custom Value")
	}
}

func TestParseTitlePageWithIndentedContinuation(t *testing.T) {
	input := "Contact:\n    Next Level Productions\n    Solvang, CA\n\nINT. X - DAY\n"
	f := Parse(input)

	if f.TitlePage.Contact != "Next Level Productions\nSolvang, CA" {
		t.Errorf("Contact = %q, want multiline value", f.TitlePage.Contact)
	}
}

func TestParseNoTitlePage(t *testing.T) {
	input := "INT. HOUSE - DAY\n"
	f := Parse(input)

	if f.TitlePage.Title != "" {
		t.Errorf("Title should be empty, got %q", f.TitlePage.Title)
	}
	if len(f.Elements) != 1 {
		t.Errorf("Elements len = %d, want 1", len(f.Elements))
	}
}

// --- Scene Heading Tests ---

func TestParseSceneHeading(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"INT. HOUSE - DAY", "INT. HOUSE - DAY"},
		{"EXT. PARK - NIGHT", "EXT. PARK - NIGHT"},
		{"EST. AIRPORT - MORNING", "EST. AIRPORT - MORNING"},
		{"INT./EXT. CAR - DAY", "INT./EXT. CAR - DAY"},
		{".FORCED HEADING", "FORCED HEADING"},
	}

	for _, tt := range tests {
		f := Parse(tt.input)
		if len(f.Elements) != 1 {
			t.Fatalf("Parse(%q): got %d elements, want 1", tt.input, len(f.Elements))
		}
		sh, ok := f.Elements[0].(SceneHeading)
		if !ok {
			t.Fatalf("Parse(%q): element type = %T, want SceneHeading", tt.input, f.Elements[0])
		}
		if sh.Text != tt.want {
			t.Errorf("Parse(%q): Text = %q, want %q", tt.input, sh.Text, tt.want)
		}
	}
}

// --- Action Tests ---

func TestParseAction(t *testing.T) {
	input := "INT. X - DAY\n\nA man walks in.\n"
	f := Parse(input)

	if len(f.Elements) != 2 {
		t.Fatalf("Elements len = %d, want 2", len(f.Elements))
	}
	action, ok := f.Elements[1].(Action)
	if !ok {
		t.Fatalf("Element type = %T, want Action", f.Elements[1])
	}
	if action.Text != "A man walks in." {
		t.Errorf("Text = %q, want %q", action.Text, "A man walks in.")
	}
}

func TestParseForcedAction(t *testing.T) {
	input := "!SCANNING THE AISLES...\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	action, ok := f.Elements[0].(Action)
	if !ok {
		t.Fatalf("Element type = %T, want Action", f.Elements[0])
	}
	if action.Text != "SCANNING THE AISLES..." {
		t.Errorf("Text = %q, want %q", action.Text, "SCANNING THE AISLES...")
	}
}

// --- Character Tests ---

func TestParseCharacter(t *testing.T) {
	input := "INT. X - DAY\n\nMAN\nHello.\n"
	f := Parse(input)

	if len(f.Elements) != 3 {
		t.Fatalf("Elements len = %d, want 3", len(f.Elements))
	}
	char, ok := f.Elements[1].(Character)
	if !ok {
		t.Fatalf("Element type = %T, want Character", f.Elements[1])
	}
	if char.Text != "MAN" {
		t.Errorf("Text = %q, want %q", char.Text, "MAN")
	}
}

func TestParseCharacterWithExtension(t *testing.T) {
	input := "INT. X - DAY\n\nMOM (O.S.)\nHello.\n"
	f := Parse(input)

	char, ok := f.Elements[1].(Character)
	if !ok {
		t.Fatalf("Element type = %T, want Character", f.Elements[1])
	}
	if char.Text != "MOM (O.S.)" {
		t.Errorf("Text = %q, want %q", char.Text, "MOM (O.S.)")
	}
}

func TestParseCharacterVO(t *testing.T) {
	input := "INT. X - DAY\n\nMADRE (V.O)\nYo siempre supe.\n"
	f := Parse(input)

	char, ok := f.Elements[1].(Character)
	if !ok {
		t.Fatalf("Element type = %T, want Character", f.Elements[1])
	}
	if char.Text != "MADRE (V.O)" {
		t.Errorf("Text = %q, want %q", char.Text, "MADRE (V.O)")
	}
}

func TestParseForcedCharacter(t *testing.T) {
	input := "@McCLANE\nYippee ki yay.\n"
	f := Parse(input)

	char, ok := f.Elements[0].(Character)
	if !ok {
		t.Fatalf("Element type = %T, want Character", f.Elements[0])
	}
	if char.Text != "McCLANE" {
		t.Errorf("Text = %q, want %q", char.Text, "McCLANE")
	}
}

// --- Dialogue Tests ---

func TestParseDialogue(t *testing.T) {
	input := "INT. X - DAY\n\nMAN\nHello there.\nHow are you?\n"
	f := Parse(input)

	// SceneHeading, Character, Dialogue, Dialogue
	if len(f.Elements) != 4 {
		t.Fatalf("Elements len = %d, want 4", len(f.Elements))
	}
	d1, ok := f.Elements[2].(Dialogue)
	if !ok {
		t.Fatalf("Element type = %T, want Dialogue", f.Elements[2])
	}
	if d1.Text != "Hello there." {
		t.Errorf("Text = %q, want %q", d1.Text, "Hello there.")
	}
	d2, ok := f.Elements[3].(Dialogue)
	if !ok {
		t.Fatalf("Element type = %T, want Dialogue", f.Elements[3])
	}
	if d2.Text != "How are you?" {
		t.Errorf("Text = %q, want %q", d2.Text, "How are you?")
	}
}

func TestParseParenthetical(t *testing.T) {
	input := "INT. X - DAY\n\nMAN\n(whispered)\nHello.\n"
	f := Parse(input)

	// SceneHeading, Character, Parenthetical, Dialogue
	if len(f.Elements) != 4 {
		t.Fatalf("Elements len = %d, want 4", len(f.Elements))
	}
	p, ok := f.Elements[2].(Parenthetical)
	if !ok {
		t.Fatalf("Element type = %T, want Parenthetical", f.Elements[2])
	}
	if p.Text != "(whispered)" {
		t.Errorf("Text = %q, want %q", p.Text, "(whispered)")
	}
}

// --- Transition Tests ---

func TestParseTransition(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CUT TO:", "CUT TO:"},
		{"SMASH CUT TO:", "SMASH CUT TO:"},
		{">Burn to White.", "Burn to White."},
	}

	for _, tt := range tests {
		f := Parse(tt.input)
		if len(f.Elements) != 1 {
			t.Fatalf("Parse(%q): got %d elements, want 1", tt.input, len(f.Elements))
		}
		tr, ok := f.Elements[0].(Transition)
		if !ok {
			t.Fatalf("Parse(%q): element type = %T, want Transition", tt.input, f.Elements[0])
		}
		if tr.Text != tt.want {
			t.Errorf("Parse(%q): Text = %q, want %q", tt.input, tr.Text, tt.want)
		}
	}
}

// --- Centered Tests ---

func TestParseCentered(t *testing.T) {
	input := ">THE END<\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	c, ok := f.Elements[0].(Centered)
	if !ok {
		t.Fatalf("Element type = %T, want Centered", f.Elements[0])
	}
	if c.Text != "THE END" {
		t.Errorf("Text = %q, want %q", c.Text, "THE END")
	}
}

// --- Lyric Tests ---

func TestParseLyric(t *testing.T) {
	input := "~Willy Wonka!\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	l, ok := f.Elements[0].(Lyric)
	if !ok {
		t.Fatalf("Element type = %T, want Lyric", f.Elements[0])
	}
	if l.Text != "Willy Wonka!" {
		t.Errorf("Text = %q, want %q", l.Text, "Willy Wonka!")
	}
}

// --- PageBreak Tests ---

func TestParsePageBreak(t *testing.T) {
	input := "INT. X - DAY\n===\nINT. Y - NIGHT\n"
	f := Parse(input)

	// SceneHeading, PageBreak, SceneHeading
	if len(f.Elements) != 3 {
		t.Fatalf("Elements len = %d, want 3", len(f.Elements))
	}
	if _, ok := f.Elements[1].(PageBreak); !ok {
		t.Fatalf("Element type = %T, want PageBreak", f.Elements[1])
	}
}

// --- Note Tests ---

func TestParseNote(t *testing.T) {
	input := "[[This is a note.]]\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	n, ok := f.Elements[0].(Note)
	if !ok {
		t.Fatalf("Element type = %T, want Note", f.Elements[0])
	}
	if n.Text != "[[This is a note.]]" {
		t.Errorf("Text = %q, want %q", n.Text, "[[This is a note.]]")
	}
}

func TestParseMultiLineNote(t *testing.T) {
	input := "[[Line 1\nLine 2\nLine 3]]\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	n, ok := f.Elements[0].(Note)
	if !ok {
		t.Fatalf("Element type = %T, want Note", f.Elements[0])
	}
	expected := "[[Line 1\nLine 2\nLine 3]]"
	if n.Text != expected {
		t.Errorf("Text = %q, want %q", n.Text, expected)
	}
}

// --- Boneyard Tests ---

func TestParseBoneyard(t *testing.T) {
	input := "/*\nThis is a comment.\n*/\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	b, ok := f.Elements[0].(Boneyard)
	if !ok {
		t.Fatalf("Element type = %T, want Boneyard", f.Elements[0])
	}
	expected := "/*\nThis is a comment.\n*/"
	if b.Text != expected {
		t.Errorf("Text = %q, want %q", b.Text, expected)
	}
}

func TestParseSingleLineBoneyard(t *testing.T) {
	input := "/* single line */\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	b, ok := f.Elements[0].(Boneyard)
	if !ok {
		t.Fatalf("Element type = %T, want Boneyard", f.Elements[0])
	}
	if b.Text != "/* single line */" {
		t.Errorf("Text = %q, want %q", b.Text, "/* single line */")
	}
}

// --- Filtering Tests ---

func TestParseFiltersEmptyLines(t *testing.T) {
	input := "INT. X - DAY\n\n\n\nA man.\n"
	f := Parse(input)

	for i, elem := range f.Elements {
		if _, ok := elem.(EmptyLine); ok {
			t.Errorf("Element[%d] is EmptyLine, should be filtered", i)
		}
	}
}

func TestParseFiltersSections(t *testing.T) {
	input := "# Act One\nINT. X - DAY\n"
	f := Parse(input)

	for i, elem := range f.Elements {
		if _, ok := elem.(Section); ok {
			t.Errorf("Element[%d] is Section, should be filtered", i)
		}
	}
}

func TestParseFiltersSynopses(t *testing.T) {
	input := "= A description.\nINT. X - DAY\n"
	f := Parse(input)

	for i, elem := range f.Elements {
		if _, ok := elem.(Synopsis); ok {
			t.Errorf("Element[%d] is Synopsis, should be filtered", i)
		}
	}
}

// --- Integration: minimal.fountain ---

func TestParseMinimalFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/minimal.fountain")
	if err != nil {
		t.Skipf("testdata file not found: %v", err)
	}

	f := Parse(string(data))

	if f.TitlePage.Title != "Test Script" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "Test Script")
	}
	if f.TitlePage.Author != "John Doe" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "John Doe")
	}

	// Expected elements: SceneHeading, Action, Character, Dialogue, Character, Parenthetical, Dialogue
	if len(f.Elements) != 7 {
		t.Fatalf("Elements len = %d, want 7", len(f.Elements))
	}

	assertType[SceneHeading](t, f.Elements[0], "SceneHeading")
	assertType[Action](t, f.Elements[1], "Action")
	assertType[Character](t, f.Elements[2], "Character")
	assertType[Dialogue](t, f.Elements[3], "Dialogue")
	assertType[Character](t, f.Elements[4], "Character")
	assertType[Parenthetical](t, f.Elements[5], "Parenthetical")
	assertType[Dialogue](t, f.Elements[6], "Dialogue")

	char := f.Elements[2].(Character)
	if char.Text != "MAN" {
		t.Errorf("Character Text = %q, want %q", char.Text, "MAN")
	}

	p := f.Elements[5].(Parenthetical)
	if p.Text != "(friendly)" {
		t.Errorf("Parenthetical Text = %q, want %q", p.Text, "(friendly)")
	}
}

// --- Integration: all-elements.fountain ---

func TestParseAllElementsFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/all-elements.fountain")
	if err != nil {
		t.Skipf("testdata file not found: %v", err)
	}

	f := Parse(string(data))

	// Title page
	if f.TitlePage.Title != "BRICK & STEEL\nFULL RETIRED" {
		t.Errorf("Title = %q, want multiline value", f.TitlePage.Title)
	}
	if f.TitlePage.Author != "Stu Maschwitz" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "Stu Maschwitz")
	}
	if f.TitlePage.Credit != "Written by" {
		t.Errorf("Credit = %q, want %q", f.TitlePage.Credit, "Written by")
	}
	if f.TitlePage.Source != "Story by KTM" {
		t.Errorf("Source = %q, want %q", f.TitlePage.Source, "Story by KTM")
	}
	if f.TitlePage.DraftDate != "1/20/2012" {
		t.Errorf("DraftDate = %q, want %q", f.TitlePage.DraftDate, "1/20/2012")
	}

	// Verify that no Section or Synopsis elements exist
	for i, elem := range f.Elements {
		if _, ok := elem.(Section); ok {
			t.Errorf("Element[%d] is Section, should be filtered", i)
		}
		if _, ok := elem.(Synopsis); ok {
			t.Errorf("Element[%d] is Synopsis, should be filtered", i)
		}
	}

	// Check for presence of key element types
	hasType := make(map[string]bool)
	for _, elem := range f.Elements {
		switch elem.(type) {
		case SceneHeading:
			hasType["SceneHeading"] = true
		case Action:
			hasType["Action"] = true
		case Character:
			hasType["Character"] = true
		case Dialogue:
			hasType["Dialogue"] = true
		case Parenthetical:
			hasType["Parenthetical"] = true
		case Transition:
			hasType["Transition"] = true
		case Centered:
			hasType["Centered"] = true
		case Note:
			hasType["Note"] = true
		case Boneyard:
			hasType["Boneyard"] = true
		case Lyric:
			hasType["Lyric"] = true
		case PageBreak:
			hasType["PageBreak"] = true
		}
	}

	required := []string{"SceneHeading", "Action", "Character", "Dialogue", "Transition", "Note", "Boneyard", "Lyric", "Centered"}
	for _, name := range required {
		if !hasType[name] {
			t.Errorf("Missing element type: %s", name)
		}
	}

	// Verify character extensions preserved literally
	for _, elem := range f.Elements {
		if ch, ok := elem.(Character); ok && ch.Text != "" {
			t.Logf("Character: %s", ch.Text)
		}
	}
}

// --- ParseFile Tests ---

func TestParseFile(t *testing.T) {
	f, err := ParseFile("../../testdata/minimal.fountain")
	if err != nil {
		t.Skipf("testdata file not found: %v", err)
	}

	if f.TitlePage.Title != "Test Script" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "Test Script")
	}
	if len(f.Elements) == 0 {
		t.Error("Expected non-empty elements")
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("nonexistent.fountain")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

// --- Empty Input ---

func TestParseEmpty(t *testing.T) {
	f := Parse("")

	if f.TitlePage.Title != "" {
		t.Errorf("Title should be empty, got %q", f.TitlePage.Title)
	}
	if len(f.Elements) != 0 {
		t.Errorf("Elements len = %d, want 0", len(f.Elements))
	}
}

// --- Dual Dialogue ---

func TestParseDualDialogue(t *testing.T) {
	input := "INT. X - DAY\n\nSTEEL\nBeer!\n\nBRICK ^\nBeer!\n"
	f := Parse(input)

	// SceneHeading, Character, Dialogue, Character, Dialogue
	if len(f.Elements) != 5 {
		t.Fatalf("Elements len = %d, want 5", len(f.Elements))
	}

	// BRICK ^ should be parsed as Character with Text "BRICK ^"
	char := f.Elements[3].(Character)
	if char.Text != "BRICK ^" {
		t.Errorf("Character Text = %q, want %q", char.Text, "BRICK ^")
	}
}

// --- Edge Case Tests ---

func TestParseDialogueWithoutCharacter(t *testing.T) {
	input := "Hello there.\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	// Without empty line before, "Hello there." is Action not Dialogue.
	_, ok := f.Elements[0].(Action)
	if !ok {
		t.Fatalf("Element type = %T, want Action (no character context)", f.Elements[0])
	}
}

func TestParseSceneHeadingOnlyDot(t *testing.T) {
	input := ".SNIPER SCOPE POV\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	sh, ok := f.Elements[0].(SceneHeading)
	if !ok {
		t.Fatalf("Element type = %T, want SceneHeading", f.Elements[0])
	}
	if sh.Text != "SNIPER SCOPE POV" {
		t.Errorf("Text = %q, want %q", sh.Text, "SNIPER SCOPE POV")
	}
}

func TestParseMultiLineBoneyard(t *testing.T) {
	input := "/*\nLine 1\nLine 2\nLine 3\nLine 4\n*/\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	b, ok := f.Elements[0].(Boneyard)
	if !ok {
		t.Fatalf("Element type = %T, want Boneyard", f.Elements[0])
	}
	expected := "/*\nLine 1\nLine 2\nLine 3\nLine 4\n*/"
	if b.Text != expected {
		t.Errorf("Text = %q, want %q", b.Text, expected)
	}
}

func TestParseMultipleConsecutiveScenes(t *testing.T) {
	input := "INT. HOUSE - DAY\n\nEXT. PARK - NIGHT\n\nINT. CAR - DAWN\n"
	f := Parse(input)

	if len(f.Elements) != 3 {
		t.Fatalf("Elements len = %d, want 3", len(f.Elements))
	}
	for i, elem := range f.Elements {
		if _, ok := elem.(SceneHeading); !ok {
			t.Errorf("Element[%d] type = %T, want SceneHeading", i, elem)
		}
	}
}

func TestParseSceneHeadingWithLowercase(t *testing.T) {
	input := "ext. house - day\n"
	f := Parse(input)

	if len(f.Elements) != 1 {
		t.Fatalf("Elements len = %d, want 1", len(f.Elements))
	}
	sh, ok := f.Elements[0].(SceneHeading)
	if !ok {
		t.Fatalf("Element type = %T, want SceneHeading", f.Elements[0])
	}
	if sh.Text != "ext. house - day" {
		t.Errorf("Text = %q, want %q", sh.Text, "ext. house - day")
	}
}

func TestParseComplexTitlePage(t *testing.T) {
	input := "Title: My Script\nCredit: Written by\nAuthor: Jane Doe\nDraft date: 01/01/2026\nContact: Studio\nSource: Novel\nRevision: v2\nCustom Key: Custom Value\n\nINT. X - DAY\n"
	f := Parse(input)

	if f.TitlePage.Title != "My Script" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "My Script")
	}
	if f.TitlePage.Credit != "Written by" {
		t.Errorf("Credit = %q, want %q", f.TitlePage.Credit, "Written by")
	}
	if f.TitlePage.Author != "Jane Doe" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "Jane Doe")
	}
	if f.TitlePage.DraftDate != "01/01/2026" {
		t.Errorf("DraftDate = %q, want %q", f.TitlePage.DraftDate, "01/01/2026")
	}
	if f.TitlePage.Contact != "Studio" {
		t.Errorf("Contact = %q, want %q", f.TitlePage.Contact, "Studio")
	}
	if f.TitlePage.Source != "Novel" {
		t.Errorf("Source = %q, want %q", f.TitlePage.Source, "Novel")
	}
	if f.TitlePage.Revision != "v2" {
		t.Errorf("Revision = %q, want %q", f.TitlePage.Revision, "v2")
	}
	if f.TitlePage.Custom["Custom Key"] != "Custom Value" {
		t.Errorf("Custom[Custom Key] = %q, want %q", f.TitlePage.Custom["Custom Key"], "Custom Value")
	}
}

func TestParseScreenplayCompleteFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/screenplay-complete.fountain")
	if err != nil {
		t.Skipf("testdata file not found: %v", err)
	}

	f := Parse(string(data))

	if f.TitlePage.Title != "THE LAST SIGNAL" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "THE LAST SIGNAL")
	}
	if f.TitlePage.Author != "Jane Smith" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "Jane Smith")
	}

	// Verify no Section or Synopsis elements in output
	for i, elem := range f.Elements {
		if _, ok := elem.(Section); ok {
			t.Errorf("Element[%d] is Section, should be filtered", i)
		}
		if _, ok := elem.(Synopsis); ok {
			t.Errorf("Element[%d] is Synopsis, should be filtered", i)
		}
	}

	// Check for presence of key element types
	hasType := make(map[string]bool)
	for _, elem := range f.Elements {
		switch elem.(type) {
		case SceneHeading:
			hasType["SceneHeading"] = true
		case Action:
			hasType["Action"] = true
		case Character:
			hasType["Character"] = true
		case Dialogue:
			hasType["Dialogue"] = true
		case Parenthetical:
			hasType["Parenthetical"] = true
		case Transition:
			hasType["Transition"] = true
		case Centered:
			hasType["Centered"] = true
		case Note:
			hasType["Note"] = true
		case Boneyard:
			hasType["Boneyard"] = true
		case Lyric:
			hasType["Lyric"] = true
		case Forced:
			hasType["Forced"] = true
		}
	}

	required := []string{"SceneHeading", "Action", "Character", "Dialogue", "Transition", "Note", "Boneyard", "Lyric", "Centered"}
	for _, name := range required {
		if !hasType[name] {
			t.Errorf("Missing element type: %s", name)
		}
	}
}

// --- Helper ---

func assertType[T Element](t *testing.T, elem Element, typeName string) {
	t.Helper()
	if _, ok := elem.(T); !ok {
		t.Errorf("Element type = %T, want %s", elem, typeName)
	}
}
