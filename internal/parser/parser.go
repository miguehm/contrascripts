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
	// Normalize YAML-style keys ("draft-date", "draft_date") to the canonical
	// space-separated Fountain key before matching.
	norm := strings.NewReplacer("-", " ", "_", " ").Replace(strings.ToLower(key))
	norm = strings.Join(strings.Fields(norm), " ")
	switch norm {
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
	case "logline":
		tp.Logline = value
	case "tagline":
		tp.Tagline = value
	case "synopsis":
		tp.Synopsis = value
	case "genre":
		tp.Genre = value
	case "subgenre", "sub genre":
		tp.Subgenre = value
	case "tone":
		tp.Tone = value
	case "themes", "theme":
		tp.Themes = value
	case "comps", "comp", "comparables":
		tp.Comps = value
	case "format":
		tp.Format = value
	case "season":
		tp.Season = value
	case "episode":
		tp.Episode = value
	case "episode title":
		tp.EpisodeTitle = value
	case "runtime", "running time":
		tp.Runtime = value
	case "pages", "page count":
		tp.Pages = value
	case "language":
		tp.Language = value
	case "status":
		tp.Status = value
	case "production company", "production co", "company":
		tp.ProdCompany = value
	case "producer":
		tp.Producer = value
	case "director", "director attached":
		tp.Director = value
	case "budget", "budget level":
		tp.Budget = value
	case "adapted by", "adaptation", "adapted":
		tp.AdaptedBy = value
	case "based on", "based":
		tp.BasedOn = value
	case "copyright", "(c)", "©":
		tp.Copyright = value
	case "wga", "wga registration", "wga number":
		tp.WGA = value
	case "rights":
		tp.Rights = value
	case "dedication", "dedicated to":
		tp.Dedication = value
	case "disclaimer":
		tp.Disclaimer = value
	case "draft type", "draft":
		tp.DraftType = value
	case "version":
		tp.Version = value
	case "revision color", "revision colour", "color":
		tp.RevisionColor = value
	case "revision date":
		tp.RevisionDate = value
	case "email", "e mail":
		tp.Email = value
	case "phone", "tel", "telephone":
		tp.Phone = value
	case "agent":
		tp.Agent = value
	case "manager":
		tp.Manager = value
	case "address":
		tp.Address = value
	case "scene numbers", "scene number", "scenenumbers", "scenenumber":
		tp.SceneNumbers = parseTitlePageBool(value)
	case "production", "production script":
		// Production scripts carry the pagination notation a spec script
		// leaves out: (MORE), (CONT'D) and the scene CONTINUED marks.
		tp.Production = parseTitlePageBool(value)
	default:
		tp.Custom[key] = value
	}
}

// parseTitlePageBool interprets a title page boolean. Accepts
// true/yes/y/on/1 (case-insensitive); anything else is false so the
// feature stays disabled by default.
func parseTitlePageBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "y", "on", "1":
		return true
	default:
		return false
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
		return Action{Text: strings.Join(p.mergeRun(TokenAction), "\n")}

	case TokenCharacter:
		p.pos++
		// A dual-dialogue caret is markup, not part of the name, so it is
		// recorded separately and kept out of the text every renderer prints.
		if name, found := strings.CutSuffix(tok.Content, "^"); found {
			return Character{Text: strings.TrimRight(name, " "), Dual: true}
		}
		return Character{Text: tok.Content}

	case TokenDialogue:
		return Dialogue{Text: strings.Join(p.mergeRun(TokenDialogue), "\n")}

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
	return Note{Text: strings.Join(p.mergeRun(TokenNote), "\n")}
}

// mergeConsecutiveBoneyards merges consecutive TokenBoneyard tokens into a single Boneyard.
func (p *Parser) mergeConsecutiveBoneyards() Boneyard {
	return Boneyard{Text: strings.Join(p.mergeRun(TokenBoneyard), "\n")}
}

// mergeRun collects the content of the run of consecutive tt tokens starting at
// the cursor. A paragraph in Fountain is a block of consecutive non-blank lines
// whose interior breaks are soft, so the run becomes one element with its
// original line structure preserved as newlines; the renderer soft wraps it.
// The run stops at the first token of any other type, which is what keeps a
// parenthetical interleaved mid-speech from being swallowed by the dialogue
// around it.
func (p *Parser) mergeRun(tt TokenType) []string {
	var parts []string
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == tt {
		parts = append(parts, p.tokens[p.pos].Content)
		p.pos++
	}
	return parts
}
