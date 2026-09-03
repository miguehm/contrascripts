package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType identifies the kind of token the lexer produces.
type TokenType int

const (
	TokenSceneHeading TokenType = iota
	TokenAction
	TokenCharacter
	TokenDialogue
	TokenParenthetical
	TokenTransition
	TokenCentered
	TokenNote
	TokenBoneyard
	TokenSection
	TokenSynopsis
	TokenLyric
	TokenForced
	TokenEmptyLine
	TokenPageBreak
)

// Token is a single classified line from the source.
type Token struct {
	Type    TokenType
	Line    int
	Content string
}

// lexerState tracks multi-line and contextual state across lines.
type lexerState struct {
	inBoneyard     bool
	inNote         bool
	expectDialogue bool
	lastTokenType  TokenType
	prevEmpty      bool
}

// Lexer tokenizes Fountain source line-by-line.
type Lexer struct {
	input  []string
	pos    int
	tokens []Token
	state  lexerState
}

// NewLexer creates a Lexer from raw Fountain source.
func NewLexer(input string) *Lexer {
	var lines []string
	if input == "" {
		lines = []string{}
	} else {
		lines = strings.Split(input, "\n")
	}
	return &Lexer{
		input:  lines,
		state:  lexerState{},
	}
}

// Tokenize processes all lines and returns the full token stream.
func (l *Lexer) Tokenize() []Token {
	l.tokens = nil
	l.pos = 0
	l.state = lexerState{}

	titleTokens, titleEnd := l.parseTitlePage()
	l.tokens = append(l.tokens, titleTokens...)
	l.pos = titleEnd

	for l.pos < len(l.input) {
		line := l.input[l.pos]
		token := l.classifyLine(line)
		l.tokens = append(l.tokens, token)
		l.state.prevEmpty = (token.Type == TokenEmptyLine)
		l.state.lastTokenType = token.Type
		l.pos++
	}

	return l.tokens
}

// parseTitlePage extracts title page metadata from the leading lines.
// Returns the title page tokens and the index of the first non-title line.
// Title pages must start with "Key: Value" lines and end at the first blank line.
func (l *Lexer) parseTitlePage() ([]Token, int) {
	if len(l.input) == 0 {
		return nil, 0
	}

	// If the very first line is blank, there is no title page.
	if strings.TrimSpace(l.input[0]) == "" {
		return nil, 0
	}

	// The first line must be a "Key: Value" line to be a title page.
	if key, value, ok := parseTitleKeyValue(strings.TrimSpace(l.input[0])); !ok || !isTitlePageEntry(key, value) {
		return nil, 0
	}

	// Scan until blank line or non-title-page line.
	titleEnd := 0
	var tokens []Token
	for i, line := range l.input {
		trimmed := strings.TrimSpace(line)

		// Blank line ends the title page.
		if trimmed == "" {
			titleEnd = i
			break
		}

		// Key: Value line.
		if key, value, ok := parseTitleKeyValue(trimmed); ok && isTitlePageEntry(key, value) {
			tokens = append(tokens, Token{
				Type:    TokenForced,
				Line:    i + 1,
				Content: key + ": " + value,
			})
			titleEnd = i + 1
			continue
		}

		// Indented continuation (3+ spaces or tab).
		if strings.HasPrefix(line, "   ") || strings.HasPrefix(line, "\t") {
			if len(tokens) > 0 {
				last := &tokens[len(tokens)-1]
				last.Content = last.Content + "\n" + trimmed
			}
			titleEnd = i + 1
			continue
		}

		// Not a title page line — stop here.
		titleEnd = i
		break
	}

	return tokens, titleEnd
}

// isTitlePageKey checks if a key looks like a valid title page key.
func isTitlePageKey(key string) bool {
	upper := strings.ToUpper(key)

	// Reject keys that are clearly not title page entries.
	rejected := []string{"CUT", "SMASH", "FADE", "CUT TO", "SMASH CUT", "SMASH CUT TO"}
	for _, r := range rejected {
		if upper == r {
			return false
		}
	}

	// Key must be non-empty and contain at least one letter.
	hasLetter := false
	for _, ch := range key {
		if unicode.IsLetter(ch) {
			hasLetter = true
			break
		}
	}

	return hasLetter
}

// isTitlePageEntry checks if a key-value pair looks like a valid title page entry.
func isTitlePageEntry(key, value string) bool {
	return isTitlePageKey(key)
}

// parseTitleKeyValue splits "Key: value" and returns (key, value, true)
// or ("", "", false) if the line does not match.
func parseTitleKeyValue(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

// classifyLine determines the token type for a single line.
func (l *Lexer) classifyLine(line string) Token {
	trimmed := strings.TrimSpace(line)

	// Empty line — context separator.
	if trimmed == "" {
		l.state.expectDialogue = false
		return Token{Type: TokenEmptyLine, Line: l.pos + 1, Content: ""}
	}

	// --- Multi-line elements: boneyard and notes ---

	// Inside a boneyard block — accumulate until closing */.
	if l.state.inBoneyard {
		return l.handleBoneyardLine(trimmed)
	}

	// Inside a note block — accumulate until closing ]].
	if l.state.inNote {
		return l.handleNoteLine(trimmed)
	}

	// --- Start of a boneyard opening on this line ---
	if strings.HasPrefix(trimmed, "/*") {
		// Check if it closes on the same line.
		if idx := strings.Index(trimmed[2:], "*/"); idx >= 0 {
			// Single-line boneyard — emit as a single token.
			return Token{Type: TokenBoneyard, Line: l.pos + 1, Content: trimmed}
		}
		// Opens a multi-line boneyard.
		l.state.inBoneyard = true
		return Token{Type: TokenBoneyard, Line: l.pos + 1, Content: trimmed}
	}

	// --- Start of a note opening on this line ---
	if strings.HasPrefix(trimmed, "[[") {
		if idx := strings.Index(trimmed[2:], "]]"); idx >= 0 {
			return Token{Type: TokenNote, Line: l.pos + 1, Content: trimmed}
		}
		l.state.inNote = true
		return Token{Type: TokenNote, Line: l.pos + 1, Content: trimmed}
	}

	// --- Page break: === (three or more equals, nothing else) ---
	if isPageBreak(trimmed) {
		return Token{Type: TokenPageBreak, Line: l.pos + 1, Content: trimmed}
	}

	// --- Section: # Heading ---
	if tok := l.trySection(trimmed); tok != nil {
		return *tok
	}

	// --- Synopsis: = Description ---
	if tok := l.trySynopsis(trimmed); tok != nil {
		return *tok
	}

	// --- Lyric: ~ text ---
	if tok := l.tryLyric(trimmed); tok != nil {
		return *tok
	}

	// --- Centered: >text< ---
	if tok := l.tryCentered(trimmed); tok != nil {
		return *tok
	}

	// --- Transition: line ending in TO: or starting with > ---
	if tok := l.tryTransition(trimmed, line); tok != nil {
		return *tok
	}

	// --- Scene Heading: .prefix or INT/EXT patterns ---
	if tok := l.trySceneHeading(trimmed); tok != nil {
		return *tok
	}

	// --- Forced Character: @NAME ---
	if tok := l.tryForcedCharacter(trimmed); tok != nil {
		l.state.expectDialogue = true
		return *tok
	}

	// --- Forced Action: !text ---
	if tok := l.tryForcedAction(trimmed); tok != nil {
		return *tok
	}

	// --- Character: UPPERCASE + preceded by empty line ---
	if tok := l.tryCharacter(trimmed); tok != nil {
		l.state.expectDialogue = true
		return *tok
	}

	// --- Parenthetical: (text) — must be checked BEFORE dialogue ---
	if tok := l.tryParenthetical(trimmed); tok != nil {
		l.state.expectDialogue = true
		return *tok
	}

	// --- Dialogue: lines after Character, Parenthetical, or Dialogue ---
	// expectDialogue stays true until an empty line resets it.
	if l.state.expectDialogue {
		return Token{Type: TokenDialogue, Line: l.pos + 1, Content: trimmed}
	}

	// --- Action: fallback ---
	return Token{Type: TokenAction, Line: l.pos + 1, Content: trimmed}
}

// handleBoneyardLine processes a line inside an open boneyard block.
func (l *Lexer) handleBoneyardLine(trimmed string) Token {
	if idx := strings.Index(trimmed, "*/"); idx >= 0 {
		l.state.inBoneyard = false
	}
	return Token{Type: TokenBoneyard, Line: l.pos + 1, Content: trimmed}
}

// handleNoteLine processes a line inside an open note block.
func (l *Lexer) handleNoteLine(trimmed string) Token {
	if idx := strings.Index(trimmed, "]]"); idx >= 0 {
		l.state.inNote = false
	}
	return Token{Type: TokenNote, Line: l.pos + 1, Content: trimmed}
}

// isPageBreak returns true if the line is three or more '=' with only whitespace.
func isPageBreak(trimmed string) bool {
	if len(trimmed) < 3 {
		return false
	}
	for _, ch := range trimmed {
		if ch != '=' {
			return false
		}
	}
	return true
}

// trySection checks for "# Heading" style sections.
func (l *Lexer) trySection(trimmed string) *Token {
	if !strings.HasPrefix(trimmed, "#") {
		return nil
	}
	level := 0
	for _, ch := range trimmed {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	if level == 0 || len(trimmed) <= level || trimmed[level] != ' ' {
		return nil
	}
	text := strings.TrimSpace(trimmed[level:])
	return &Token{Type: TokenSection, Line: l.pos + 1, Content: text}
}

// trySynopsis checks for "= Description" lines.
func (l *Lexer) trySynopsis(trimmed string) *Token {
	if !strings.HasPrefix(trimmed, "=") {
		return nil
	}
	if len(trimmed) < 2 || trimmed[1] != ' ' {
		return nil
	}
	text := strings.TrimSpace(trimmed[1:])
	return &Token{Type: TokenSynopsis, Line: l.pos + 1, Content: text}
}

// tryLyric checks for "~ text" lines.
func (l *Lexer) tryLyric(trimmed string) *Token {
	if !strings.HasPrefix(trimmed, "~") {
		return nil
	}
	text := strings.TrimSpace(trimmed[1:])
	return &Token{Type: TokenLyric, Line: l.pos + 1, Content: text}
}

// tryCentered checks for ">text<" centered text.
func (l *Lexer) tryCentered(trimmed string) *Token {
	if !strings.HasPrefix(trimmed, ">") || !strings.HasSuffix(trimmed, "<") {
		return nil
	}
	text := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	return &Token{Type: TokenCentered, Line: l.pos + 1, Content: text}
}

// tryTransition checks for transition lines (ending in TO: or forced with >).
func (l *Lexer) tryTransition(trimmed, rawLine string) *Token {
	// Forced transition: > at start (but not >text< centered).
	if strings.HasPrefix(trimmed, ">") && !strings.HasSuffix(trimmed, "<") {
		text := strings.TrimSpace(trimmed[1:])
		return &Token{Type: TokenTransition, Line: l.pos + 1, Content: text}
	}

	// Natural transition: ALL CAPS ending in TO:
	if strings.HasSuffix(trimmed, "TO:") && isAllUpper(trimmed[:len(trimmed)-3]) {
		return &Token{Type: TokenTransition, Line: l.pos + 1, Content: trimmed}
	}

	return nil
}

// trySceneHeading checks for scene headings.
func (l *Lexer) trySceneHeading(trimmed string) *Token {
	// Forced scene heading: . prefix followed by alphanumeric.
	if len(trimmed) > 1 && trimmed[0] == '.' {
		nextRune, _ := utf8.DecodeRuneInString(trimmed[1:])
		if unicode.IsLetter(nextRune) || unicode.IsDigit(nextRune) {
			text := trimmed[1:]
			return &Token{Type: TokenSceneHeading, Line: l.pos + 1, Content: text}
		}
	}

	// Natural scene heading: INT, EXT, etc. patterns (case-insensitive).
	upper := strings.ToUpper(trimmed)
	prefixes := []string{"INT.", "EXT.", "EST.", "INT./EXT.", "INT/EXT.", "I/E."}
	for _, pfx := range prefixes {
		if strings.HasPrefix(upper, pfx) {
			return &Token{Type: TokenSceneHeading, Line: l.pos + 1, Content: trimmed}
		}
	}

	// Also match "INT " or "EXT " (space after prefix).
	spacePrefixes := []string{"INT ", "EXT ", "EST ", "INT./EXT ", "INT/EXT ", "I/E "}
	for _, pfx := range spacePrefixes {
		if strings.HasPrefix(upper, pfx) {
			return &Token{Type: TokenSceneHeading, Line: l.pos + 1, Content: trimmed}
		}
	}

	return nil
}

// tryForcedCharacter checks for "@NAME" lines.
func (l *Lexer) tryForcedCharacter(trimmed string) *Token {
	if !strings.HasPrefix(trimmed, "@") {
		return nil
	}
	name := strings.TrimSpace(trimmed[1:])
	if name == "" {
		return nil
	}
	return &Token{Type: TokenCharacter, Line: l.pos + 1, Content: name}
}

// tryForcedAction checks for "!text" lines.
func (l *Lexer) tryForcedAction(trimmed string) *Token {
	if !strings.HasPrefix(trimmed, "!") {
		return nil
	}
	text := strings.TrimSpace(trimmed[1:])
	return &Token{Type: TokenAction, Line: l.pos + 1, Content: text}
}

// tryCharacter checks for UPPERCASE character names preceded by an empty line.
func (l *Lexer) tryCharacter(trimmed string) *Token {
	// Must be preceded by an empty line.
	if !l.state.prevEmpty {
		return nil
	}

	// Must be entirely uppercase (allowing extensions in parens).
	if !isCharacterName(trimmed) {
		return nil
	}

	return &Token{Type: TokenCharacter, Line: l.pos + 1, Content: trimmed}
}

// tryParenthetical checks for "(text)" lines.
func (l *Lexer) tryParenthetical(trimmed string) *Token {
	if !strings.HasPrefix(trimmed, "(") || !strings.HasSuffix(trimmed, ")") {
		return nil
	}
	if len(trimmed) < 3 {
		return nil
	}
	return &Token{Type: TokenParenthetical, Line: l.pos + 1, Content: trimmed}
}

// isAllUpper returns true if s contains at least one letter and all letters are uppercase.
func isAllUpper(s string) bool {
	hasLetter := false
	for _, ch := range s {
		if unicode.IsLetter(ch) {
			hasLetter = true
			if unicode.IsLower(ch) {
				return false
			}
		}
	}
	return hasLetter
}

// isCharacterName checks if the line is a valid character name:
// at least one letter, mostly uppercase, with optional extension in parens.
func isCharacterName(s string) bool {
	// Strip parenthetical extension if present.
	name := s
	if idx := strings.Index(s, "("); idx >= 0 {
		name = strings.TrimSpace(s[:idx])
	}

	if name == "" {
		return false
	}

	// Must contain at least one letter.
	hasLetter := false
	upperCount := 0
	totalLetters := 0

	for _, ch := range name {
		if unicode.IsLetter(ch) {
			hasLetter = true
			totalLetters++
			if unicode.IsUpper(ch) {
				upperCount++
			}
		}
	}

	if !hasLetter {
		return false
	}

	// All letters must be uppercase for a natural character name.
	return upperCount == totalLetters
}
