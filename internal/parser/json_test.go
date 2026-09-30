package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToDocumentInlineAndUppercaseFlags(t *testing.T) {
	f := Parse("EXT. CASA - DIA\n" +
		"\n" +
		"Silencio **total**.\n" +
		"\n" +
		"ELENA\n" +
		"(murmura)\n" +
		"Can you _hear_ me?\n" +
		"\n" +
		"CUT TO:\n" +
		"\n" +
		">ADIOS<\n" +
		"\n" +
		"~una cancion\n" +
		"\n" +
		"@BRICK\n" +
		"Hola\n" +
		"\n" +
		".FORCED SCENE\n" +
		"\n" +
		"!FORCED ACTION\n")

	doc := ToDocument(f)
	byType := map[string]Node{}
	for _, n := range doc.Elements {
		if _, dup := byType[n.Type]; !dup {
			byType[n.Type] = n
		}
	}

	tests := []struct {
		typ        string
		wantUpper  bool
		wantItalic bool
		wantInline bool
	}{
		{"sceneHeading", true, false, false},
		{"action", false, false, true},
		{"character", true, false, false},
		{"parenthetical", false, false, true},
		{"dialogue", false, false, true},
		{"transition", true, false, false},
		{"centered", false, false, true},
		{"lyric", false, true, true},
	}

	for _, tt := range tests {
		n, ok := byType[tt.typ]
		if !ok {
			t.Errorf("no %q element in document", tt.typ)
			continue
		}
		if n.Uppercase != tt.wantUpper {
			t.Errorf("%s: Uppercase = %v, want %v", tt.typ, n.Uppercase, tt.wantUpper)
		}
		if n.BaseItalic != tt.wantItalic {
			t.Errorf("%s: BaseItalic = %v, want %v", tt.typ, n.BaseItalic, tt.wantItalic)
		}
		if got := len(n.Inline) > 0; got != tt.wantInline {
			t.Errorf("%s: has inline = %v, want %v", tt.typ, got, tt.wantInline)
		}
	}
}

func TestToDocumentForcedPrefixes(t *testing.T) {
	f := Parse("@BRICK\nHola\n\n.FORCED SCENE\n\n!FORCED ACTION\n")

	var got []string
	for _, n := range ToDocument(f).Elements {
		got = append(got, n.Type)
	}

	want := []string{"character", "dialogue", "sceneHeading", "action"}
	if len(got) != len(want) {
		t.Fatalf("elements = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("element %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestToDocumentNestedInline(t *testing.T) {
	f := Parse("**bold _and italic_** tail.\n")
	nodes := ToDocument(f).Elements
	if len(nodes) != 1 || len(nodes[0].Inline) == 0 {
		t.Fatalf("expected one element with inline spans, got %+v", nodes)
	}

	bold := nodes[0].Inline[0]
	if bold.Type != "bold" {
		t.Fatalf("first span = %q, want bold", bold.Type)
	}
	if len(bold.Content) != 2 {
		t.Fatalf("bold content = %d spans, want 2", len(bold.Content))
	}
	if bold.Content[0].Text != "bold " {
		t.Errorf("bold text = %q, want %q", bold.Content[0].Text, "bold ")
	}
	inner := bold.Content[1]
	if inner.Type != "italic" {
		t.Errorf("nested span = %q, want italic", inner.Type)
	}
	if len(inner.Content) != 1 || inner.Content[0].Text != "and italic" {
		t.Errorf("italic content = %+v, want text %q", inner.Content, "and italic")
	}
	if tail := nodes[0].Inline[1]; tail.Type != "text" || tail.Text != " tail." {
		t.Errorf("trailing span = %+v, want text %q", tail, " tail.")
	}
}

func TestToDocumentDropsEmptyLines(t *testing.T) {
	f := Parse("EXT. CASA - DIA\n\nAction.\n\n")
	for _, n := range ToDocument(f).Elements {
		if n.Type == "" {
			t.Fatalf("empty node leaked into document: %+v", n)
		}
	}
}

func TestToDocumentNilSafe(t *testing.T) {
	if got := ToDocument(nil); got == nil || len(got.Elements) != 0 {
		t.Errorf("ToDocument(nil) = %+v, want empty document", got)
	}
}

func TestToJSONUsesLowercaseKeys(t *testing.T) {
	b := ToJSON(Parse("title: T\n\nEXT. CASA - DIA\n"))

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, b)
	}
	for _, key := range []string{"titlePage", "elements"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing key %q in %s", key, b)
		}
	}
}

// TestToJSONTitlePageKeysAreLowerCamel guards the wire format. TitlePage once
// shipped capitalised keys while every other type used lowerCamel, which a
// JavaScript consumer cannot discover by reading the Go struct.
func TestToJSONTitlePageKeysAreLowerCamel(t *testing.T) {
	b := ToJSON(Parse("title: T\ncustom-key: v\n\nEXT. CASA - DIA\n"))

	var raw struct {
		TitlePage map[string]json.RawMessage `json:"titlePage"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, b)
	}
	if len(raw.TitlePage) == 0 {
		t.Fatal("titlePage is empty")
	}

	want := []string{"title", "author", "draftDate", "prodCompany", "wga", "sceneNumbers", "custom"}
	for _, key := range want {
		if _, ok := raw.TitlePage[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
	for key := range raw.TitlePage {
		if key[0] >= 'A' && key[0] <= 'Z' {
			t.Errorf("key %q is not lowerCamel", key)
		}
	}
}

func TestToJSONCustomIsAlwaysAnObject(t *testing.T) {
	// Custom must never encode as null: JavaScript callers index it directly.
	f := &Fountain{Elements: []Element{Action{Text: "x"}}}
	if got := string(ToJSON(f)); !strings.Contains(got, `"custom":{}`) {
		t.Errorf("nil Custom encoded as %s, want an empty object", got)
	}
}

// A clean script must encode an empty array, never null, so a consumer can
// iterate the result without guarding. See ToWarningsJSON.
func TestToWarningsJSONEmptyIsAnArrayNotNull(t *testing.T) {
	for _, input := range []string{"", "\n\n", "Title: T\nAuthor: A\n\nEXT. CASA - DIA\n\nTexto.\n"} {
		if got := string(ToWarningsJSON(input)); got != "[]" {
			t.Errorf("ToWarningsJSON(%q) = %s, want []", input, got)
		}
	}
}

func TestToWarningsJSONUsesLowercaseKeys(t *testing.T) {
	b := ToWarningsJSON("EXT. CASA - DIA\n")

	var out []map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, b)
	}
	if len(out) == 0 {
		t.Skip("lint produced no warnings for this input")
	}
	for _, key := range []string{"line", "code", "message"} {
		if _, ok := out[0][key]; !ok {
			t.Errorf("missing key %q in %s", key, b)
		}
	}
}

// TestToJSONMatchesEncodingJSON pins our hand-rolled encoder against the
// standard library, so the WebAssembly wire format cannot drift from what
// a consumer would get from encoding/json.
func TestToJSONMatchesEncodingJSON(t *testing.T) {
	f := Parse("Title: La señal\n" +
		"Author:\n  - Sarahi\n  - Miguel\n" +
		"zeta: last\n" +
		"alpha: first\n" +
		"---\n" +
		"\n" +
		"EXT. CASA - DIA\n" +
		"\n" +
		"Says **bold _and italic_** now.\n" +
		"\n" +
		"ELENA\n" +
		"¿Sí? \"quoted\"\n" +
		"\n" +
		">ADIOS<\n" +
		"\n" +
		"~canción\tcon tabulador\n" +
		"\n" +
		"CUT TO:\n" +
		"\n" +
		"=== \n" +
		"\n" +
		"!FORCED\n")

	doc := ToDocument(f)
	want, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("encoding/json: %v", err)
	}
	if got := ToJSON(f); string(got) != string(want) {
		t.Errorf("hand-rolled encoder diverges from encoding/json\n got: %s\nwant: %s", got, want)
	}
}

func TestAppendJSONEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hola", `"hola"`},
		{"quote", `say "hi"`, `"say \"hi\""`},
		{"backslash", `a\b`, `"a\\b"`},
		{"newline", "a\nb", `"a\nb"`},
		{"tab", "a\tb", `"a\tb"`},
		{"control", "a\x01b", `"a\u0001b"`},
		{"unicode", "LA SEÑAL", `"LA SEÑAL"`},
		{"emoji", "🕐", `"🕐"`},
		{"invalid utf8", "a\xffb", `"a\ufffdb"`},
		{"empty", "", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(appendJSONString(nil, tt.in))
			if got != tt.want {
				t.Errorf("appendJSONString(%q) = %s, want %s", tt.in, got, tt.want)
			}
			if !json.Valid([]byte(got)) {
				t.Errorf("output is not valid JSON: %s", got)
			}
		})
	}
}

func TestAppendJSONOmitempty(t *testing.T) {
	type sample struct {
		Kept    string `json:"kept"`
		Dropped string `json:"dropped,omitempty"`
		Number  int    `json:"number,omitempty"`
		Flag    bool   `json:"flag,omitempty"`
		hidden  string
	}

	got := string(appendJSON(nil, sample{Kept: "x", hidden: "y"}))
	if got != `{"kept":"x"}` {
		t.Errorf("appendJSON = %s, want %s", got, `{"kept":"x"}`)
	}
}

func TestAppendJSONMapSortsKeys(t *testing.T) {
	m := map[string]int{"zebra": 3, "alpha": 1, "middle": 2}
	got := string(appendJSON(nil, m))
	if got != `{"alpha":1,"middle":2,"zebra":3}` {
		t.Errorf("appendJSON map = %s, want sorted keys", got)
	}
}

// TestDualDialogueOnTheWire checks the caret is not lost to the wire format. The
// parser takes it off the cue text, so without a field of its own the fact that a
// cue was marked dual would simply disappear for anything reading the JSON.
func TestDualDialogueOnTheWire(t *testing.T) {
	dual := ToJSON(Parse("\nINT. HOUSE - DAY\n\nBRICK ^\nBeer!\n"))
	if !strings.Contains(string(dual), `"dual":true`) {
		t.Errorf("ToJSON(dual cue) = %s, want a dual field", dual)
	}
	if !strings.Contains(string(dual), `"text":"BRICK"`) {
		t.Errorf("ToJSON(dual cue) = %s, want the name without the caret", dual)
	}

	plain := ToJSON(Parse("\nINT. HOUSE - DAY\n\nBRICK\nBeer!\n"))
	if strings.Contains(string(plain), "dual") {
		t.Errorf("ToJSON(plain cue) = %s, want no dual field so the wire is unchanged for scripts that do not use it", plain)
	}
}
