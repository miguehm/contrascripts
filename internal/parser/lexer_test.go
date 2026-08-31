package parser

import (
	"os"
	"testing"
)

func tokenTypes(tokens []Token) []TokenType {
	types := make([]TokenType, len(tokens))
	for i, t := range tokens {
		types[i] = t.Type
	}
	return types
}

func assertTokenType(t *testing.T, got TokenType, want TokenType, line int) {
	t.Helper()
	if got != want {
		t.Errorf("line %d: got token type %d, want %d", line, got, want)
	}
}

func assertTokenContent(t *testing.T, got, want string, line int) {
	t.Helper()
	if got != want {
		t.Errorf("line %d: got content %q, want %q", line, got, want)
	}
}

func TestEmptyInput(t *testing.T) {
	lexer := NewLexer("")
	tokens := lexer.Tokenize()
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

func TestTitlePage(t *testing.T) {
	input := "Title: My Script\nAuthor: John Doe\n\nINT. HOUSE - DAY"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
	}

	// Title page tokens are emitted as TokenForced with "Key: Value" content.
	if tokens[0].Type != TokenForced {
		t.Errorf("token 0: got type %d, want TokenForced", tokens[0].Type)
	}
	if tokens[0].Content != "Title: My Script" {
		t.Errorf("token 0: got content %q, want %q", tokens[0].Content, "Title: My Script")
	}
	if tokens[1].Content != "Author: John Doe" {
		t.Errorf("token 1: got content %q, want %q", tokens[1].Content, "Author: John Doe")
	}

	// After title page, there should be a SceneHeading.
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenSceneHeading {
			found = true
			if tok.Content != "INT. HOUSE - DAY" {
				t.Errorf("scene heading: got content %q, want %q", tok.Content, "INT. HOUSE - DAY")
			}
			break
		}
	}
	if !found {
		t.Error("expected SceneHeading token after title page")
	}
}

func TestSceneHeading(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"INT. HOUSE - DAY", "INT. HOUSE - DAY"},
		{"EXT. BRICK'S POOL - DAY", "EXT. BRICK'S POOL - DAY"},
		{"EST. PARK - MORNING", "EST. PARK - MORNING"},
		{"INT./EXT. CAR - NIGHT", "INT./EXT. CAR - NIGHT"},
		{"INT/EXT. BOAT - DAY", "INT/EXT. BOAT - DAY"},
		{"I/E. PLANE - DUSK", "I/E. PLANE - DUSK"},
		{".FORCED SCENE", "FORCED SCENE"},
		{"ext. house - day", "ext. house - day"},
	}

	for _, tt := range tests {
		// Add empty line before to reset context.
		input := "\n" + tt.input
		lexer := NewLexer(input)
		tokens := lexer.Tokenize()

		found := false
		for _, tok := range tokens {
			if tok.Type == TokenSceneHeading {
				found = true
				if tok.Content != tt.want {
					t.Errorf("input %q: got content %q, want %q", tt.input, tok.Content, tt.want)
				}
				break
			}
		}
		if !found {
			t.Errorf("input %q: no SceneHeading token found", tt.input)
		}
	}
}

func TestAction(t *testing.T) {
	input := "\nA man enters the room."
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	found := false
	for _, tok := range tokens {
		if tok.Type == TokenAction {
			found = true
			if tok.Content != "A man enters the room." {
				t.Errorf("got content %q, want %q", tok.Content, "A man enters the room.")
			}
			break
		}
	}
	if !found {
		t.Error("expected Action token")
	}
}

func TestCharacter(t *testing.T) {
	input := "\nMAN\nHello there."
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var charToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenCharacter {
			charToken = &tokens[i]
			break
		}
	}

	if charToken == nil {
		t.Fatal("expected Character token")
	}
	if charToken.Content != "MAN" {
		t.Errorf("got content %q, want %q", charToken.Content, "MAN")
	}
}

func TestCharacterWithExtension(t *testing.T) {
	input := "\nMOM (O.S.)\nCome down for supper!"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var charToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenCharacter {
			charToken = &tokens[i]
			break
		}
	}

	if charToken == nil {
		t.Fatal("expected Character token")
	}
	if charToken.Content != "MOM (O.S.)" {
		t.Errorf("got content %q, want %q", charToken.Content, "MOM (O.S.)")
	}
}

func TestDialogue(t *testing.T) {
	input := "\nMAN\nHello there."
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var dialogToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenDialogue {
			dialogToken = &tokens[i]
			break
		}
	}

	if dialogToken == nil {
		t.Fatal("expected Dialogue token")
	}
	if dialogToken.Content != "Hello there." {
		t.Errorf("got content %q, want %q", dialogToken.Content, "Hello there.")
	}
}

func TestParenthetical(t *testing.T) {
	input := "\nMAN\n(whispered)\nHello."
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var parenToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenParenthetical {
			parenToken = &tokens[i]
			break
		}
	}

	if parenToken == nil {
		t.Fatal("expected Parenthetical token")
	}
	if parenToken.Content != "(whispered)" {
		t.Errorf("got content %q, want %q", parenToken.Content, "(whispered)")
	}
}

func TestTransition(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CUT TO:", "CUT TO:"},
		{"SMASH CUT TO:", "SMASH CUT TO:"},
		{"FADE TO:", "FADE TO:"},
	}

	for _, tt := range tests {
		// Transitions need empty lines before and after.
		input := "\n" + tt.input + "\n"
		lexer := NewLexer(input)
		tokens := lexer.Tokenize()

		found := false
		for _, tok := range tokens {
			if tok.Type == TokenTransition {
				found = true
				if tok.Content != tt.want {
					t.Errorf("input %q: got content %q, want %q", tt.input, tok.Content, tt.want)
				}
				break
			}
		}
		if !found {
			t.Errorf("input %q: no Transition token found", tt.input)
		}
	}
}

func TestForcedTransition(t *testing.T) {
	input := "\n>Burn to White.\n"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var transToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenTransition {
			transToken = &tokens[i]
			break
		}
	}

	if transToken == nil {
		t.Fatal("expected Transition token")
	}
	if transToken.Content != "Burn to White." {
		t.Errorf("got content %q, want %q", transToken.Content, "Burn to White.")
	}
}

func TestCentered(t *testing.T) {
	input := "\n>THE END<"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var centToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenCentered {
			centToken = &tokens[i]
			break
		}
	}

	if centToken == nil {
		t.Fatal("expected Centered token")
	}
	if centToken.Content != "THE END" {
		t.Errorf("got content %q, want %q", centToken.Content, "THE END")
	}
}

func TestBoneyard(t *testing.T) {
	input := "/*\nThis is commented out.\nINT. HOUSE - DAY\n*/\n\nEXT. PARK - DAY"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	boneyardCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenBoneyard {
			boneyardCount++
		}
	}

	// Should have 4 boneyard lines (/*, content, content, */).
	if boneyardCount != 4 {
		t.Errorf("expected 4 boneyard tokens, got %d", boneyardCount)
	}

	// The "INT. HOUSE - DAY" inside boneyard should NOT be a SceneHeading.
	// But "EXT. PARK - DAY" after the boneyard SHOULD be a SceneHeading.
	foundSceneAfterBoneyard := false
	for _, tok := range tokens {
		if tok.Type == TokenSceneHeading && tok.Content == "EXT. PARK - DAY" {
			foundSceneAfterBoneyard = true
		}
	}
	if !foundSceneAfterBoneyard {
		t.Error("expected SceneHeading for 'EXT. PARK - DAY' after boneyard")
	}
}

func TestNote(t *testing.T) {
	input := "\n[[This is a note.]]\n\nINT. HOUSE - DAY"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	noteCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenNote {
			noteCount++
		}
	}

	if noteCount != 1 {
		t.Errorf("expected 1 note token, got %d", noteCount)
	}
}

func TestMultiLineNote(t *testing.T) {
	input := "\n[[Line 1\nLine 2\nLine 3]]\n\nINT. HOUSE - DAY"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	noteCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenNote {
			noteCount++
		}
	}

	// Should have 3 note tokens (opening, content, closing).
	if noteCount != 3 {
		t.Errorf("expected 3 note tokens, got %d", noteCount)
	}
}

func TestSection(t *testing.T) {
	tests := []struct {
		input string
		level int
		text  string
	}{
		{"# Act One", 1, "Act One"},
		{"## Sequence", 2, "Sequence"},
		{"### Scene", 3, "Scene"},
	}

	for _, tt := range tests {
		lexer := NewLexer(tt.input)
		tokens := lexer.Tokenize()

		found := false
		for _, tok := range tokens {
			if tok.Type == TokenSection {
				found = true
				if tok.Content != tt.text {
					t.Errorf("input %q: got content %q, want %q", tt.input, tok.Content, tt.text)
				}
				break
			}
		}
		if !found {
			t.Errorf("input %q: no Section token found", tt.input)
		}
	}
}

func TestSynopsis(t *testing.T) {
	input := "= Set up the characters and the story."
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var synToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenSynopsis {
			synToken = &tokens[i]
			break
		}
	}

	if synToken == nil {
		t.Fatal("expected Synopsis token")
	}
	if synToken.Content != "Set up the characters and the story." {
		t.Errorf("got content %q, want %q", synToken.Content, "Set up the characters and the story.")
	}
}

func TestLyric(t *testing.T) {
	input := "~Willy Wonka! Willy Wonka!"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var lyricToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenLyric {
			lyricToken = &tokens[i]
			break
		}
	}

	if lyricToken == nil {
		t.Fatal("expected Lyric token")
	}
	if lyricToken.Content != "Willy Wonka! Willy Wonka!" {
		t.Errorf("got content %q, want %q", lyricToken.Content, "Willy Wonka! Willy Wonka!")
	}
}

func TestForcedCharacter(t *testing.T) {
	input := "\n@McCLANE\nYippie ki-yay!"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var charToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenCharacter {
			charToken = &tokens[i]
			break
		}
	}

	if charToken == nil {
		t.Fatal("expected Character token")
	}
	if charToken.Content != "McCLANE" {
		t.Errorf("got content %q, want %q", charToken.Content, "McCLANE")
	}
}

func TestForcedAction(t *testing.T) {
	input := "\n!SCANNING THE AISLES..."
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var actToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenAction {
			actToken = &tokens[i]
			break
		}
	}

	if actToken == nil {
		t.Fatal("expected Action token")
	}
	if actToken.Content != "SCANNING THE AISLES..." {
		t.Errorf("got content %q, want %q", actToken.Content, "SCANNING THE AISLES...")
	}
}

func TestPageBreak(t *testing.T) {
	input := "===\n"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	var pbToken *Token
	for i, tok := range tokens {
		if tok.Type == TokenPageBreak {
			pbToken = &tokens[i]
			break
		}
	}

	if pbToken == nil {
		t.Fatal("expected PageBreak token")
	}
}

func TestEmptyLine(t *testing.T) {
	input := "Hello\n\nWorld"
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	emptyCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenEmptyLine {
			emptyCount++
		}
	}

	if emptyCount != 1 {
		t.Errorf("expected 1 EmptyLine token, got %d", emptyCount)
	}
}

func TestDualDialogue(t *testing.T) {
	input := "\nBRICK\nScrew retirement.\n\nSTEEL ^\nScrew retirement."
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	charCount := 0
	for _, tok := range tokens {
		if tok.Type == TokenCharacter {
			charCount++
		}
	}

	if charCount != 2 {
		t.Errorf("expected 2 Character tokens, got %d", charCount)
	}
}

func TestMinimalFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/minimal.fountain")
	if err != nil {
		t.Skipf("testdata file not found: %v", err)
	}

	lexer := NewLexer(string(data))
	tokens := lexer.Tokenize()

	// Verify we get expected token types.
	types := tokenTypes(tokens)

	// Must contain title page tokens, scene heading, action, character, dialogue.
	hasScene := false
	hasChar := false
	hasDialog := false
	for _, typ := range types {
		switch typ {
		case TokenSceneHeading:
			hasScene = true
		case TokenCharacter:
			hasChar = true
		case TokenDialogue:
			hasDialog = true
		}
	}

	if !hasScene {
		t.Error("minimal.fountain: missing SceneHeading")
	}
	if !hasChar {
		t.Error("minimal.fountain: missing Character")
	}
	if !hasDialog {
		t.Error("minimal.fountain: missing Dialogue")
	}
}

func TestAllElementsFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/all-elements.fountain")
	if err != nil {
		t.Skipf("testdata file not found: %v", err)
	}

	lexer := NewLexer(string(data))
	tokens := lexer.Tokenize()

	// Collect all token types present.
	present := make(map[TokenType]bool)
	for _, tok := range tokens {
		present[tok.Type] = true
	}

	// These types should be present in all-elements.fountain.
	required := []struct {
		typ  TokenType
		name string
	}{
		{TokenSceneHeading, "SceneHeading"},
		{TokenAction, "Action"},
		{TokenCharacter, "Character"},
		{TokenDialogue, "Dialogue"},
		{TokenParenthetical, "Parenthetical"},
		{TokenTransition, "Transition"},
		{TokenCentered, "Centered"},
		{TokenNote, "Note"},
		{TokenBoneyard, "Boneyard"},
		{TokenSection, "Section"},
		{TokenSynopsis, "Synopsis"},
		{TokenLyric, "Lyric"},
		{TokenEmptyLine, "EmptyLine"},
	}

	for _, req := range required {
		if !present[req.typ] {
			t.Errorf("all-elements.fountain: missing token type %s", req.name)
		}
	}
}

func TestIsAllUpper(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"HELLO", true},
		{"Hello", false},
		{"hello", false},
		{"CUT TO:", true},
		{"123", false},
		{"", false},
		{"A", true},
	}

	for _, tt := range tests {
		got := isAllUpper(tt.input)
		if got != tt.want {
			t.Errorf("isAllUpper(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsCharacterName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"MAN", true},
		{"BRICK", true},
		{"MOM (O.S.)", true},
		{"HANS (on the radio)", true},
		{"man", false},
		{"Man", false},
		{"123", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isCharacterName(tt.input)
		if got != tt.want {
			t.Errorf("isCharacterName(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
