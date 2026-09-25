package parser

import (
	"strings"
	"unicode"
)

type Warning struct {
	Line    int    `json:"line"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Lint(input string) []Warning {
	var out []Warning
	if input == "" {
		return nil
	}
	lines := strings.Split(input, "\n")
	tokens := NewLexer(input).Tokenize()
	bodyStart := titleBodyStart(lines)

	lineToken := map[int]TokenType{}
	for _, tok := range tokens {
		if tok.Type == TokenForced || tok.Type == TokenEmptyLine {
			continue
		}
		if _, ok := lineToken[tok.Line]; !ok {
			lineToken[tok.Line] = tok.Type
		}
	}

	out = append(out, lintHeader(lines)...)
	out = append(out, lintUnclosedBlocks(lines)...)
	out = append(out, lintInlineLines(lines, bodyStart, lineToken)...)
	out = append(out, lintTokens(tokens, lines)...)
	out = append(out, lintCounts(tokens)...)
	out = append(out, lintNames(tokens)...)

	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && (out[j].Line < out[j-1].Line || (out[j].Line == out[j-1].Line && out[j].Code < out[j-1].Code)) {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}

func titleBodyStart(lines []string) int {
	if len(lines) == 0 {
		return 1
	}
	if strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				return i + 2
			}
		}
		return 1
	}
	if _, _, ok := parseTitleKeyValue(strings.TrimSpace(lines[0])); !ok {
		return 1
	}
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || trimmed == "---" {
			break
		}
		if _, _, ok := parseTitleKeyValue(trimmed); ok {
			i++
			continue
		}
		if strings.HasPrefix(lines[i], " ") || strings.HasPrefix(lines[i], "\t") || strings.HasPrefix(trimmed, "- ") {
			i++
			continue
		}
		break
	}
	return i + 1
}

func lintHeader(lines []string) []Warning {
	var out []Warning
	if len(lines) == 0 {
		return nil
	}
	yaml := strings.TrimSpace(lines[0]) == "---"
	entries := map[string]string{}
	order := []string{}
	if yaml {
		closed := false
		current := ""
		for i := 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "---" {
				closed = true
				break
			}
			if trimmed == "" {
				continue
			}
			if k, v, ok := parseTitleKeyValue(trimmed); ok {
				current = strings.ToLower(k)
				v = stripTitleQuotes(v)
				entries[current] = v
				order = append(order, current)
				continue
			}
			if current != "" {
				cont := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if entries[current] == "" {
					entries[current] = cont
				} else {
					entries[current] += "\n" + cont
				}
			}
		}
		if !closed {
			return []Warning{{Line: 1, Code: "UNCLOSED_FRONTMATTER", Message: "YAML frontmatter opened with --- but never closed"}}
		}
	} else {
		if _, _, ok := parseTitleKeyValue(strings.TrimSpace(lines[0])); !ok {
			return nil
		}
		last := ""
		for i := 0; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" || trimmed == "---" {
				break
			}
			if k, v, ok := parseTitleKeyValue(trimmed); ok {
				lk := strings.ToLower(k)
				entries[lk] = v
				order = append(order, lk)
				last = lk
				continue
			}
			if last != "" && (strings.HasPrefix(lines[i], " ") || strings.HasPrefix(lines[i], "\t") || strings.HasPrefix(trimmed, "- ")) {
				cont := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if entries[last] == "" {
					entries[last] = cont
				} else {
					entries[last] += "\n" + cont
				}
				continue
			}
			break
		}
	}
	norm := func(k string) string {
		n := strings.NewReplacer("-", " ", "_", " ").Replace(strings.ToLower(k))
		return strings.Join(strings.Fields(n), " ")
	}
	byNorm := map[string]string{}
	for k, v := range entries {
		byNorm[norm(k)] = v
	}
	_ = order
	if _, ok := byNorm["title"]; !ok {
		out = append(out, Warning{Line: 1, Code: "TITLE_MISSING", Message: "title page has no Title entry"})
	}
	if _, ok := byNorm["author"]; !ok {
		out = append(out, Warning{Line: 1, Code: "AUTHOR_MISSING", Message: "title page has no Author entry"})
	}
	if c, ok := byNorm["revision color"]; ok && strings.TrimSpace(c) != "" {
		switch strings.ToLower(strings.TrimSpace(c)) {
		case "white", "blue", "pink", "yellow", "green", "goldenrod":
		default:
			out = append(out, Warning{Line: 1, Code: "BAD_REVISION_COLOR", Message: "revision-color should be white|blue|pink|yellow|green|goldenrod, got " + strings.TrimSpace(c)})
		}
	}
	for k, v := range entries {
		if norm(k) == "scene numbers" || norm(k) == "scene number" || k == "scenenumbers" || k == "scenenumber" {
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "yes", "y", "on", "1", "false", "no", "n", "off", "0", "":
			default:
				out = append(out, Warning{Line: 1, Code: "BAD_SCENE_NUMBERS", Message: "scene-numbers accepts true/yes/y/on/1, got " + strings.TrimSpace(v)})
			}
		}
	}
	return out
}

func lintUnclosedBlocks(lines []string) []Warning {
	var out []Warning
	inBoneyard := false
	boneOpen := 0
	inNote := false
	noteOpen := 0
	for i, raw := range lines {
		ln := i + 1
		trimmed := strings.TrimSpace(raw)
		if !inBoneyard && !inNote {
			if strings.HasPrefix(trimmed, "/*") {
				rest := trimmed[2:]
				if idx := strings.Index(rest, "*/"); idx < 0 {
					inBoneyard = true
					boneOpen = ln
				}
				continue
			}
			if strings.HasPrefix(trimmed, "[[") {
				rest := trimmed[2:]
				if idx := strings.Index(rest, "]]"); idx < 0 {
					inNote = true
					noteOpen = ln
				}
				continue
			}
			continue
		}
		if inBoneyard {
			if strings.Contains(trimmed, "*/") {
				inBoneyard = false
			}
			continue
		}
		if inNote {
			if strings.Contains(trimmed, "]]") {
				inNote = false
			}
		}
	}
	if inBoneyard {
		out = append(out, Warning{Line: boneOpen, Code: "UNCLOSED_BONEYARD", Message: "boneyard opened with /* but never closed with */"})
	}
	if inNote {
		out = append(out, Warning{Line: noteOpen, Code: "UNCLOSED_NOTE", Message: "note opened with [[ but never closed with ]]"})
	}
	return out
}

func lintInlineLines(lines []string, bodyStart int, lineToken map[int]TokenType) []Warning {
	var out []Warning
	for i, raw := range lines {
		ln := i + 1
		if ln < bodyStart {
			continue
		}
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if typ, ok := lineToken[ln]; ok && (typ == TokenBoneyard || typ == TokenNote) {
			continue
		}
		if strings.Count(raw, "**")%2 == 1 {
			out = append(out, Warning{Line: ln, Code: "UNCLOSED_BOLD", Message: "odd number of ** markers, bold may not be closed"})
			continue
		}
		if indexItalic(raw) >= 0 || indexSingleItalic(raw) >= 0 {
			parsed := ParseInline(raw)
			for _, el := range parsed {
				if pt, ok := el.(PlainText); ok && (strings.Contains(pt.Text, "_") || hasStrayStar(pt.Text)) {
					out = append(out, Warning{Line: ln, Code: "UNCLOSED_ITALIC", Message: "italic marker _ or * may not be closed"})
					break
				}
			}
		}
	}
	return out
}

func hasStrayStar(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '*' {
			continue
		}
		if i+1 < len(s) && s[i+1] == '*' {
			i++
			continue
		}
		if i > 0 && s[i-1] == '*' {
			continue
		}
		return true
	}
	return false
}

func lintTokens(tokens []Token, lines []string) []Warning {
	var out []Warning
	rawOf := func(ln int) string {
		if ln >= 1 && ln <= len(lines) {
			return strings.TrimSpace(lines[ln-1])
		}
		return ""
	}
	sig := []Token{}
	for _, tok := range tokens {
		if tok.Type == TokenForced {
			continue
		}
		sig = append(sig, tok)
	}
	nextSig := func(idx int) *Token {
		for i := idx + 1; i < len(sig); i++ {
			switch sig[i].Type {
			case TokenEmptyLine, TokenSection, TokenSynopsis, TokenNote, TokenBoneyard, TokenPageBreak:
				continue
			default:
				return &sig[i]
			}
		}
		return nil
	}
	prevSig := func(idx int) *Token {
		for i := idx - 1; i >= 0; i-- {
			switch sig[i].Type {
			case TokenEmptyLine, TokenSection, TokenSynopsis, TokenNote, TokenBoneyard, TokenPageBreak:
				continue
			default:
				return &sig[i]
			}
		}
		return nil
	}
	seenScene := false
	for i, tok := range sig {
		switch tok.Type {
		case TokenSceneHeading:
			seenScene = true
			n := nextSig(i)
			if n == nil || n.Type == TokenSceneHeading {
				out = append(out, Warning{Line: tok.Line, Code: "SCENE_EMPTY", Message: "scene heading has no content before next heading"})
			}
			if !strings.Contains(tok.Content, " - ") && !strings.HasPrefix(rawOf(tok.Line), ".") {
				out = append(out, Warning{Line: tok.Line, Code: "SCENE_NO_TIME", Message: "scene heading usually ends with location - TIME (e.g. - DAY)"})
			}
		case TokenCharacter:
			if !seenScene {
				out = append(out, Warning{Line: tok.Line, Code: "DIALOG_BEFORE_SCENE", Message: "character cue appears before first scene heading"})
			}
			n := nextSig(i)
			if n == nil || (n.Type != TokenDialogue && n.Type != TokenParenthetical) {
				out = append(out, Warning{Line: tok.Line, Code: "CHARACTER_NO_DIALOGUE", Message: "character cue has no dialogue after it"})
			}
			text := strings.TrimSpace(tok.Content)
			upperText := strings.ToUpper(text)
			if strings.HasSuffix(upperText, " TO") {
				out = append(out, Warning{Line: tok.Line, Code: "POSSIBLE_TRANSITION", Message: "character cue looks like a transition missing colon (TO:)"})
			}
			if possibleScene(text) {
				out = append(out, Warning{Line: tok.Line, Code: "POSSIBLE_SCENE", Message: "character cue looks like a scene heading, needs . or space after INT/EXT/EST"})
			}
			if strings.Contains(text, "^") && !strings.HasSuffix(text, "^") {
				out = append(out, Warning{Line: tok.Line, Code: "DUAL_MISMATCH", Message: "dual-dialogue caret ^ must be the last character on the line"})
			} else if strings.HasSuffix(text, "^") {
				base := strings.TrimSpace(strings.TrimSuffix(text, "^"))
				if base == "" {
					out = append(out, Warning{Line: tok.Line, Code: "EMPTY_FORCED", Message: "dual-dialogue marker ^ has no character name"})
				}
			}
			if idx := strings.Index(text, "("); idx >= 0 {
				after := text[idx:]
				if !strings.Contains(after, ")") {
					out = append(out, Warning{Line: tok.Line, Code: "BAD_EXTENSION", Message: "character extension is missing closing )"})
				} else {
					inner := after[strings.Index(after, "(")+1 : strings.Index(after, ")")]
					if len(strings.TrimSpace(inner)) > 30 {
						out = append(out, Warning{Line: tok.Line, Code: "BAD_EXTENSION", Message: "character extension looks too long: (" + strings.TrimSpace(inner) + ")"})
					}
				}
			}
		case TokenDialogue:
			if !seenScene {
				out = append(out, Warning{Line: tok.Line, Code: "DIALOG_BEFORE_SCENE", Message: "dialogue appears before first scene heading"})
			}
			p := prevSig(i)
			if p == nil || (p.Type != TokenCharacter && p.Type != TokenDialogue && p.Type != TokenParenthetical) {
				out = append(out, Warning{Line: tok.Line, Code: "ORPHAN_DIALOGUE", Message: "dialogue has no character cue before it"})
			}
		case TokenParenthetical:
			p := prevSig(i)
			if p == nil || (p.Type != TokenCharacter && p.Type != TokenDialogue && p.Type != TokenParenthetical) {
				out = append(out, Warning{Line: tok.Line, Code: "ORPHAN_PAREN", Message: "parenthetical has no character cue before it"})
			}
			if len(tok.Content) > 60 {
				out = append(out, Warning{Line: tok.Line, Code: "LONG_PAREN", Message: "parenthetical is long, keep direction brief"})
			}
		case TokenTransition:
			if strings.TrimSpace(tok.Content) == "" {
				out = append(out, Warning{Line: tok.Line, Code: "EMPTY_FORCED", Message: "transition marker > has no text"})
			}
			n := nextSig(i)
			if n != nil && n.Type == TokenTransition {
				out = append(out, Warning{Line: tok.Line, Code: "TRANSITION_CHAIN", Message: "two transitions in a row, one is probably unintended"})
			}
		case TokenCentered:
			if strings.TrimSpace(tok.Content) == "" {
				out = append(out, Warning{Line: tok.Line, Code: "EMPTY_FORCED", Message: "centered text >...< is empty"})
			}
		case TokenLyric:
			if strings.TrimSpace(tok.Content) == "" {
				out = append(out, Warning{Line: tok.Line, Code: "EMPTY_FORCED", Message: "lyric marker ~ has no text"})
			}
		case TokenAction:
			if strings.TrimSpace(tok.Content) == "" {
				if rawOf(tok.Line) == "!" {
					out = append(out, Warning{Line: tok.Line, Code: "EMPTY_FORCED", Message: "forced prefix ! has no text"})
				}
				break
			}
			if raw := rawOf(tok.Line); (strings.HasPrefix(raw, "!") || strings.HasPrefix(raw, "@")) && len(strings.TrimSpace(tok.Content)) > 1 {
				break
			}
			out = append(out, lintActionLine(tok)...)
		}
	}
	return out
}

func lintActionLine(tok Token) []Warning {
	var out []Warning
	content := tok.Content
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	if trimmed == "@" || trimmed == "!" || trimmed == "." || trimmed == "~" || trimmed == ">" {
		out = append(out, Warning{Line: tok.Line, Code: "EMPTY_FORCED", Message: "forced prefix " + trimmed + " has no text"})
		return out
	}
	if strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "!") {
		return out
	}
	upper := strings.ToUpper(trimmed)
	if strings.HasSuffix(upper, "TO:") && !isAllUpper(strings.TrimSpace(trimmed[:len(trimmed)-3])) {
		out = append(out, Warning{Line: tok.Line, Code: "POSSIBLE_TRANSITION", Message: "line ends with TO: but is not uppercase, transitions must be UPPERCASE"})
		return out
	}
	if isAllUpper(trimmed) && strings.HasSuffix(strings.TrimSpace(trimmed), "TO") && !strings.HasSuffix(strings.TrimSpace(trimmed), "TO:") {
		out = append(out, Warning{Line: tok.Line, Code: "POSSIBLE_TRANSITION", Message: "all-caps line ends with TO, did you mean TO:?"})
		return out
	}
	if possibleScene(trimmed) {
		out = append(out, Warning{Line: tok.Line, Code: "POSSIBLE_SCENE", Message: "line starts with INT/EXT/EST but is not a scene heading, needs . or space after prefix"})
		return out
	}
	if isCharacterName(trimmed) || (isAllUpper(trimmed) && hasLetter(trimmed)) {
		out = append(out, Warning{Line: tok.Line, Code: "POSSIBLE_CHARACTER", Message: "all-caps line parsed as action, add blank line before for character or ! to force action"})
		return out
	}
	if strings.HasPrefix(trimmed, "#") {
		out = append(out, Warning{Line: tok.Line, Code: "BAD_SECTION", Message: "sections need # plus a space (# Title)"})
		return out
	}
	if strings.HasPrefix(trimmed, "=") && !strings.HasPrefix(trimmed, "==") {
		out = append(out, Warning{Line: tok.Line, Code: "BAD_SYNOPSIS", Message: "synopses need = plus a space (= Text)"})
		return out
	}
	if len(trimmed) >= 3 {
		eq := true
		for _, ch := range trimmed {
			if ch != '=' {
				eq = false
				break
			}
		}
		if !eq && strings.HasPrefix(trimmed, "===") {
			out = append(out, Warning{Line: tok.Line, Code: "BAD_PAGEBREAK", Message: "page breaks must be only === with nothing else"})
			return out
		}
	}
	return out
}

func hasLetter(s string) bool {
	for _, ch := range s {
		if unicode.IsLetter(ch) {
			return true
		}
	}
	return false
}

func possibleScene(trimmed string) bool {
	upper := strings.ToUpper(trimmed)
	cands := []string{"INT", "EXT", "EST", "I/E", "INT/EXT", "INT./EXT"}
	for _, c := range cands {
		if upper == c {
			return true
		}
		if strings.HasPrefix(upper, c) {
			rest := trimmed[len(c):]
			if rest == "" {
				return true
			}
			r := rest[0]
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				continue
			}
			if r == '-' || r == '/' || r == ':' || r == '_' || r == '>' || r == '.' || r == '#' {
				return true
			}
		}
	}
	return false
}

func lintCounts(tokens []Token) []Warning {
	var out []Warning
	centered, lyrics, breaks, trans := 0, 0, 0, 0
	centerLine, lyricLine, breakLine := 0, 0, 0
	actionRun, actionStart := 0, 0
	dialogRun, dialogStart := 0, 0
	flushRun := func() {
		if actionRun > 4 {
			out = append(out, Warning{Line: actionStart, Code: "LONG_ACTION", Message: "action block is long, consider splitting"})
		}
		if dialogRun > 4 {
			out = append(out, Warning{Line: dialogStart, Code: "LONG_DIALOGUE", Message: "dialogue block is long, consider splitting"})
		}
		actionRun, dialogRun = 0, 0
	}
	for _, tok := range tokens {
		switch tok.Type {
		case TokenAction:
			if actionRun == 0 {
				actionStart = tok.Line
			}
			actionRun++
			dialogRun = 0
		case TokenDialogue:
			if dialogRun == 0 {
				dialogStart = tok.Line
			}
			dialogRun++
			actionRun = 0
		case TokenEmptyLine:
			flushRun()
		default:
			flushRun()
		}
		switch tok.Type {
		case TokenCentered:
			centered++
			if centerLine == 0 {
				centerLine = tok.Line
			}
		case TokenLyric:
			lyrics++
			if lyricLine == 0 {
				lyricLine = tok.Line
			}
		case TokenPageBreak:
			breaks++
			if breakLine == 0 {
				breakLine = tok.Line
			}
		case TokenTransition:
			trans++
		}
	}
	flushRun()
	if centered > 5 {
		out = append(out, Warning{Line: centerLine, Code: "OVERUSE_CENTERED", Message: "many centered lines, use sparingly"})
	}
	if lyrics > 5 {
		out = append(out, Warning{Line: lyricLine, Code: "OVERUSE_LYRIC", Message: "many lyric lines"})
	}
	if breaks > 3 {
		out = append(out, Warning{Line: breakLine, Code: "OVERUSE_PAGEBREAK", Message: "many === page breaks"})
	}
	if trans > 10 {
		out = append(out, Warning{Line: 1, Code: "OVERUSE_TRANSITION", Message: "many transitions, CUT TO: sparingly"})
	}
	return out
}

func lintNames(tokens []Token) []Warning {
	var out []Warning
	type nameSeen struct {
		raw  string
		line int
	}
	byUpper := map[string][]nameSeen{}
	uniqBase := []string{}
	seenBase := map[string]bool{}
	stripName := func(raw string) string {
		u := strings.ToUpper(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "^")))
		if idx := strings.Index(u, "("); idx >= 0 {
			u = strings.TrimSpace(u[:idx])
		}
		return u
	}
	for _, tok := range tokens {
		if tok.Type != TokenCharacter {
			continue
		}
		raw := strings.TrimSpace(tok.Content)
		base := stripName(raw)
		if base == "" {
			continue
		}
		byUpper[base] = append(byUpper[base], nameSeen{raw: raw, line: tok.Line})
		if !seenBase[base] {
			seenBase[base] = true
			uniqBase = append(uniqBase, base)
		}
	}
	for base, list := range byUpper {
		cased := map[string]map[string]bool{}
		firstLine := list[0].line
		for _, s := range list {
			keep := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s.raw), "^"))
			if idx := strings.Index(keep, "("); idx >= 0 {
				keep = strings.TrimSpace(keep[:idx])
			}
			up := strings.ToUpper(keep)
			if cased[up] == nil {
				cased[up] = map[string]bool{}
			}
			cased[up][keep] = true
		}
		for up, variants := range cased {
			if len(variants) > 1 {
				names := []string{}
				for v := range variants {
					names = append(names, v)
				}
				out = append(out, Warning{Line: firstLine, Code: "NAME_INCONSISTENT", Message: "character " + up + " written with different casing: " + strings.Join(names, " / ")})
			}
		}
		_ = base
	}
	for i := 0; i < len(uniqBase); i++ {
		for j := i + 1; j < len(uniqBase); j++ {
			a, b := uniqBase[i], uniqBase[j]
			if a == b {
				continue
			}
			if editDistance(a, b) > 0 {
				d := editDistance(a, b)
				minLen := len([]rune(a))
				if lb := len([]rune(b)); lb < minLen {
					minLen = lb
				}
				if (d == 1 && minLen >= 4) || (d == 2 && minLen >= 7) {
					out = append(out, Warning{Line: byUpper[b][0].line, Code: "NAME_INCONSISTENT", Message: "similar character names " + a + " / " + b + ", possible typo"})
				}
			}
		}
	}
	return out
}

func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		cur := make([]int, len(br)+1)
		cur[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			cur[j] = prev[j] + 1
			if cur[j-1]+1 < cur[j] {
				cur[j] = cur[j-1] + 1
			}
			if prev[j-1]+cost < cur[j] {
				cur[j] = prev[j-1] + cost
			}
		}
		prev = cur
	}
	return prev[len(br)]
}
