package parser

import (
	"os"
	"strings"
)

// Parse converts raw Fountain text into a Fountain AST.
func Parse(input string) *Fountain {
	tokens := NewLexer(input).Tokenize()
	p := &Parser{tokens: tokens}
	return p.parse()
}

// ParseFile reads a Fountain file and parses it into a Fountain AST.
func ParseFile(path string) (*Fountain, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data)), nil
}

// Parser holds state during parsing.
type Parser struct {
	tokens []Token
	pos    int
}

// parse builds the full Fountain AST from the token stream.
func (p *Parser) parse() *Fountain {
	f := &Fountain{}

	f.TitlePage = p.parseTitlePage()

	for p.pos < len(p.tokens) {
		elem := p.nextElement()
		if elem != nil {
			f.Elements = append(f.Elements, elem)
		}
	}

	return f
}

// parseTitlePage extracts title page metadata from leading TokenForced tokens.
func (p *Parser) parseTitlePage() TitlePage {
	tp := TitlePage{
		Custom: make(map[string]string),
	}

	for p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		if tok.Type != TokenForced {
			break
		}

		key, value, ok := parseTitleKeyValue(tok.Content)
		if !ok {
			break
		}

		// Validate the key looks like a real title page key, not a transition.
		if !isTitlePageKey(key) {
			break
		}

		p.pos++
		p.assignTitleField(&tp, key, value)
	}

	return tp
}

// assignTitleField sets a known title page field or falls back to Custom.
func (p *Parser) assignTitleField(tp *TitlePage, key, value string) {
	switch strings.ToLower(key) {
	case "title":
		tp.Title = value
	case "credit":
		tp.Credit = value
	case "author":
		tp.Author = value
	case "draft date":
		tp.DraftDate = value
	case "contact":
		tp.Contact = value
	case "source":
		tp.Source = value
	case "revision":
		tp.Revision = value
	default:
		tp.Custom[key] = value
	}
}

// nextElement reads the next element from the token stream.
// Returns nil for tokens that are filtered out (EmptyLine, Section, Synopsis).
func (p *Parser) nextElement() Element {
	if p.pos >= len(p.tokens) {
		return nil
	}

	tok := p.tokens[p.pos]

	switch tok.Type {
	case TokenEmptyLine:
		p.pos++
		return nil

	case TokenSection:
		p.pos++
		return nil

	case TokenSynopsis:
		p.pos++
		return nil

	case TokenSceneHeading:
		p.pos++
		return SceneHeading{Text: tok.Content}

	case TokenAction:
		p.pos++
		return Action{Text: tok.Content}

	case TokenCharacter:
		p.pos++
		name, ext := parseCharacterName(tok.Content)
		return Character{Name: name, Extension: ext}

	case TokenDialogue:
		p.pos++
		return Dialogue{Text: tok.Content}

	case TokenParenthetical:
		p.pos++
		return Parenthetical{Text: tok.Content}

	case TokenTransition:
		p.pos++
		return Transition{Text: tok.Content}

	case TokenCentered:
		p.pos++
		return Centered{Text: tok.Content}

	case TokenNote:
		return p.mergeConsecutiveNotes()

	case TokenBoneyard:
		return p.mergeConsecutiveBoneyards()

	case TokenLyric:
		p.pos++
		return Lyric{Text: tok.Content}

	case TokenPageBreak:
		p.pos++
		return PageBreak{}

	case TokenForced:
		p.pos++
		return Forced{Prefix: '.', Text: tok.Content}

	default:
		p.pos++
		return nil
	}
}

// mergeConsecutiveNotes merges consecutive TokenNote tokens into a single Note.
func (p *Parser) mergeConsecutiveNotes() Note {
	var parts []string
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenNote {
		parts = append(parts, p.tokens[p.pos].Content)
		p.pos++
	}
	return Note{Text: strings.Join(parts, "\n")}
}

// mergeConsecutiveBoneyards merges consecutive TokenBoneyard tokens into a single Boneyard.
func (p *Parser) mergeConsecutiveBoneyards() Boneyard {
	var parts []string
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenBoneyard {
		parts = append(parts, p.tokens[p.pos].Content)
		p.pos++
	}
	return Boneyard{Text: strings.Join(parts, "\n")}
}

// parseCharacterName splits "NAME (EXT)" into name and extension.
func parseCharacterName(content string) (name, extension string) {
	idx := strings.Index(content, "(")
	if idx < 0 {
		return content, ""
	}

	name = strings.TrimSpace(content[:idx])
	ext := content[idx+1:]
	ext = strings.TrimSuffix(ext, ")")
	ext = strings.TrimSpace(ext)

	return name, ext
}
