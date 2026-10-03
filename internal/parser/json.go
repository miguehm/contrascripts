package parser

// Document is the JSON-friendly projection of a parsed Fountain script.
//
// The AST uses interface-typed nodes (Element, InlineElement), which
// encoding/json cannot marshal on its own, and renderers need per-node
// hints (uppercasing, inline support, base italic) that the AST does not
// carry. Document is the stable wire format consumed by non-Go renderers
// such as the web preview.
type Document struct {
	TitlePage      TitlePage      `json:"titlePage"`
	TitlePageLines map[string]int `json:"titlePageLines,omitempty"`
	Elements       []Node         `json:"elements"`
}

// Node is the JSON projection of a single screenplay Element.
type Node struct {
	// Type is the lowercase element name, e.g. "action" or "sceneHeading".
	Type string `json:"type"`
	// Line is the 1-based source line where the element starts.
	Line int `json:"line"`
	// Text is the element text exactly as parsed, before inline parsing.
	Text string `json:"text,omitempty"`
	// Level is the heading depth, set for sections only.
	Level int `json:"level,omitempty"`
	// Prefix is the forcing character ('.', '!', '@'), set for forced elements.
	Prefix string `json:"prefix,omitempty"`
	// Uppercase reports that renderers must upper-case Text to match the PDF.
	Uppercase bool `json:"uppercase,omitempty"`
	// BaseItalic reports that renderers must style the whole element italic.
	BaseItalic bool `json:"baseItalic,omitempty"`
	// Dual reports that a character cue carried the dual-dialogue caret, which the
	// parser strips from Text, so only this records that it was there.
	Dual bool `json:"dual,omitempty"`
	// Inline holds the parsed inline formatting, present only for element
	// types where the PDF renderer honours **bold** and *italic*.
	Inline []Span `json:"inline,omitempty"`
}

// Span is the JSON projection of a single InlineElement.
type Span struct {
	// Type is "text", "bold" or "italic".
	Type string `json:"type"`
	// Text is the literal run, set for type "text".
	Text string `json:"text,omitempty"`
	// Content holds nested spans, set for "bold" and "italic".
	Content []Span `json:"content,omitempty"`
}

// ToDocument projects a parsed script into its JSON-friendly form.
func ToDocument(doc *Fountain) *Document {
	if doc == nil {
		return &Document{}
	}
	tp := doc.TitlePage
	if tp.Custom == nil {
		tp.Custom = map[string]string{}
	}
	out := &Document{TitlePage: tp, TitlePageLines: doc.TitlePageLines, Elements: make([]Node, 0, len(doc.Elements))}
	for _, el := range doc.Elements {
		if node, ok := toNode(el); ok {
			out.Elements = append(out.Elements, node)
		}
	}
	return out
}

// ToJSON marshals a parsed script to its JSON wire format. It cannot fail.
func ToJSON(doc *Fountain) []byte {
	return appendJSON(nil, ToDocument(doc))
}

// ToWarningsJSON marshals lint warnings to their JSON wire format.
//
// A clean script yields an empty array, never null: consumers iterate the
// result without a guard, and ToDocument already sets the same expectation for
// Document.elements.
func ToWarningsJSON(input string) []byte {
	warnings := Lint(input)
	if warnings == nil {
		warnings = []Warning{}
	}
	return appendJSON(nil, warnings)
}

// ToNode projects a single element, reporting false for types that carry no
// renderable payload (empty lines, notes, boneyards).
func toNode(el Element) (Node, bool) {
	switch e := el.(type) {
	case SceneHeading:
		return node("sceneHeading", e.Line, e.Text, true, false, false), true
	case Action:
		return node("action", e.Line, e.Text, false, false, true), true
	case Character:
		return dualNode("character", e.Line, e.Text, e.Dual), true
	case Dialogue:
		return node("dialogue", e.Line, e.Text, false, false, true), true
	case Parenthetical:
		return node("parenthetical", e.Line, e.Text, false, false, true), true
	case Transition:
		return node("transition", e.Line, e.Text, true, false, false), true
	case Centered:
		return node("centered", e.Line, e.Text, false, false, true), true
	case Lyric:
		return node("lyric", e.Line, e.Text, false, true, true), true
	case Forced:
		return toForced(e), true
	case Note:
		return node("note", e.Line, e.Text, false, false, false), true
	case Boneyard:
		return node("boneyard", e.Line, e.Text, false, false, false), true
	case Section:
		return Node{Type: "section", Line: e.Line, Text: e.Text, Level: e.Level}, true
	case Synopsis:
		return node("synopsis", e.Line, e.Text, false, false, false), true
	case PageBreak:
		return Node{Type: "pageBreak", Line: e.Line}, true
	case EmptyLine:
		return Node{}, false
	default:
		return Node{}, false
	}
}

// toForced resolves a forced element to the type its prefix maps to, so the
// JSON mirrors what the PDF actually draws rather than exposing the prefix.
func toForced(e Forced) Node {
	switch e.Prefix {
	case '.':
		return node("sceneHeading", e.Line, e.Text, true, false, false)
	case '@':
		return node("character", e.Line, e.Text, true, false, false)
	case '!':
		return node("action", e.Line, e.Text, false, false, true)
	default:
		return node("action", e.Line, e.Text, false, false, false)
	}
}

// dualNode is node for a character cue, carrying the dual-dialogue marker. The
// caret is stripped from the text by the parser, so without this the fact that a
// cue was marked dual would be lost to anything reading the wire format.
func dualNode(typ string, line int, text string, dual bool) Node {
	n := node(typ, line, text, true, false, false)
	n.Dual = dual
	return n
}

func node(typ string, line int, text string, uppercase, baseItalic, inline bool) Node {
	n := Node{Type: typ, Line: line, Text: text, Uppercase: uppercase, BaseItalic: baseItalic}
	if inline {
		n.Inline = spans(ParseInline(text))
	}
	return n
}

func spans(inlines []InlineElement) []Span {
	if len(inlines) == 0 {
		return nil
	}
	out := make([]Span, 0, len(inlines))
	for _, in := range inlines {
		switch e := in.(type) {
		case PlainText:
			out = append(out, Span{Type: "text", Text: e.Text})
		case Bold:
			out = append(out, Span{Type: "bold", Content: spans(e.Content)})
		case Italic:
			out = append(out, Span{Type: "italic", Content: spans(e.Content)})
		}
	}
	return out
}
