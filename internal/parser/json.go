package parser

// Document is the JSON-friendly projection of a parsed Fountain script.
//
// The AST uses interface-typed nodes (Element, InlineElement), which
// encoding/json cannot marshal on its own, and renderers need per-node
// hints (uppercasing, inline support, base italic) that the AST does not
// carry. Document is the stable wire format consumed by non-Go renderers
// such as the web preview.
type Document struct {
	TitlePage TitlePage `json:"titlePage"`
	Elements  []Node    `json:"elements"`
}

// Node is the JSON projection of a single screenplay Element.
type Node struct {
	// Type is the lowercase element name, e.g. "action" or "sceneHeading".
	Type string `json:"type"`
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
	out := &Document{TitlePage: tp, Elements: make([]Node, 0, len(doc.Elements))}
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
func ToWarningsJSON(input string) []byte {
	return appendJSON(nil, Lint(input))
}

// ToNode projects a single element, reporting false for types that carry no
// renderable payload (empty lines, notes, boneyards).
func toNode(el Element) (Node, bool) {
	switch e := el.(type) {
	case SceneHeading:
		return node("sceneHeading", e.Text, true, false, false), true
	case Action:
		return node("action", e.Text, false, false, true), true
	case Character:
		return node("character", e.Text, true, false, false), true
	case Dialogue:
		return node("dialogue", e.Text, false, false, true), true
	case Parenthetical:
		return node("parenthetical", e.Text, false, false, true), true
	case Transition:
		return node("transition", e.Text, true, false, false), true
	case Centered:
		return node("centered", e.Text, false, false, true), true
	case Lyric:
		return node("lyric", e.Text, false, true, true), true
	case Forced:
		return toForced(e), true
	case Note:
		return node("note", e.Text, false, false, false), true
	case Boneyard:
		return node("boneyard", e.Text, false, false, false), true
	case Section:
		return Node{Type: "section", Text: e.Text, Level: e.Level}, true
	case Synopsis:
		return node("synopsis", e.Text, false, false, false), true
	case PageBreak:
		return Node{Type: "pageBreak"}, true
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
		return node("sceneHeading", e.Text, true, false, false)
	case '@':
		return node("character", e.Text, true, false, false)
	case '!':
		return node("action", e.Text, false, false, true)
	default:
		return node("action", e.Text, false, false, false)
	}
}

func node(typ, text string, uppercase, baseItalic, inline bool) Node {
	n := Node{Type: typ, Text: text, Uppercase: uppercase, BaseItalic: baseItalic}
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
