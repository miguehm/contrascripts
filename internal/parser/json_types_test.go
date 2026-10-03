package parser

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The TypeScript declarations in web/fountain.d.ts are hand-written, and
// nothing compiles them against this wire format: the .d.ts ships to a browser
// project that has no Go toolchain, and nothing imports it from here. So this
// test is the only thing standing between a new TitlePage field and a
// JavaScript consumer that reads `undefined` for it.
//
// It reads the .d.ts from the module root rather than a copy, so the file
// `make dist` ships is the file that was checked.

const typesFile = "../../web/fountain.d.ts"

// titlePageProperty matches one property declaration inside the TitlePage
// interface of the .d.ts, e.g. `  draftDate: string` or `  custom?: Foo`.
var titlePageProperty = regexp.MustCompile(`(?m)^\s*(\w+)\??\s*:`)

func TestTypesFileDeclaresEveryTitlePageField(t *testing.T) {
	raw, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatalf("read %s: %v", typesFile, err)
	}
	declared := titlePageFields(t, string(raw))

	rt := reflect.TypeFor[TitlePage]()
	for i := 0; i < rt.NumField(); i++ {
		name := jsonTagName(rt.Field(i))
		if name == "-" {
			continue
		}
		if !declared[name] {
			t.Errorf("TitlePage.%s is on the wire but not in %s; add it, or a TypeScript consumer reads undefined", name, typesFile)
		}
	}
}

// TestTypesFileHasNoUnknownTitlePageFields is the other direction: a name in
// the .d.ts that this struct does not emit is a typo, and it is invisible from
// the browser side, where every missing key looks the same as an absent one.
func TestTypesFileHasNoUnknownTitlePageFields(t *testing.T) {
	raw, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatalf("read %s: %v", typesFile, err)
	}
	declared := titlePageFields(t, string(raw))

	emitted := map[string]bool{}
	rt := reflect.TypeFor[TitlePage]()
	for i := 0; i < rt.NumField(); i++ {
		emitted[jsonTagName(rt.Field(i))] = true
	}

	for name := range declared {
		if !emitted[name] {
			t.Errorf("%s declares titlePage.%s, which TitlePage never emits", typesFile, name)
		}
	}
}

// titlePageFields returns the property names declared by the TitlePage
// interface in the .d.ts, failing the test if the interface is not there or
// cannot be located — a silent empty set would make both tests above vacuous.
func titlePageFields(t *testing.T, src string) map[string]bool {
	t.Helper()

	const open = "export interface TitlePage {"
	start := strings.Index(src, open)
	if start < 0 {
		t.Fatalf("%s has no %q block", typesFile, strings.TrimSpace(open))
	}
	body := src[start+len(open):]
	end := strings.Index(body, "\n}")
	if end < 0 {
		t.Fatalf("%s: unterminated TitlePage interface", typesFile)
	}

	fields := map[string]bool{}
	for _, match := range titlePageProperty.FindAllStringSubmatch(body[:end], -1) {
		fields[match[1]] = true
	}
	if len(fields) == 0 {
		t.Fatalf("%s: TitlePage interface declares no properties", typesFile)
	}
	return fields
}

// TestTitlePageFieldsAreAlwaysEmitted documents why the .d.ts declares every
// field as required rather than optional: the wire format has no omitempty on
// TitlePage, so an unset key is an empty string, not a missing one.
func TestTitlePageFieldsAreAlwaysEmitted(t *testing.T) {
	rt := reflect.TypeFor[TitlePage]()
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		if strings.Contains(tag, "omitempty") {
			t.Errorf("TitlePage.%s has omitempty; the .d.ts says every field is always present", rt.Field(i).Name)
		}
	}
}

func TestTypesFileListsEveryElementType(t *testing.T) {
	raw, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatalf("read %s: %v", typesFile, err)
	}
	src := string(raw)

	// The element names the wire format can produce, collected from a parse
	// that exercises every branch of toNode rather than from a constant, so a
	// new type added there without a union member here is caught.
	doc := ToDocument(Parse(strings.Join([]string{
		"Title: T",
		"Author: A",
		"",
		"# Section",
		"## Sub",
		"= Synopsis",
		"[[Note]]",
		"/* boneyard */",
		"===",
		"EXT. CASA - DIA",
		"",
		"Acción.",
		"",
		"ELENA",
		"(murmura)",
		"Hola.",
		"",
		">CENTRADO<",
		"",
		"~canción",
		"",
		"CUT TO:",
		"",
		"@BRICK",
		"!FORCED",
		".FORCED SCENE",
	}, "\n")))

	seen := map[string]bool{}
	for _, n := range doc.Elements {
		seen[n.Type] = true
	}
	if len(seen) == 0 {
		t.Fatal("no elements parsed; the fixture no longer covers anything")
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !strings.Contains(src, "'"+name+"'") {
			t.Errorf("element type %q is not in the ElementType union in %s", name, typesFile)
		}
	}
}

// TestWireFormatCarriesSourceLines pins the two fields this change adds:
// every element node carries its 1-based source line, and recognized title
// page keys map to their source line. A regression that drops either would
// leave the double-click-to-source feature dead without a compile error.
func TestWireFormatCarriesSourceLines(t *testing.T) {
	input := "Title: T\nDraft-Date: hoy\n\nEXT. CASA - DIA\n\nAcción.\n"
	doc := ToDocument(Parse(input))

	if doc.TitlePageLines["title"] != 1 {
		t.Errorf("titlePageLines[title] = %d, want 1", doc.TitlePageLines["title"])
	}
	if doc.TitlePageLines["draft date"] != 2 {
		t.Errorf("titlePageLines[draft date] = %d, want 2 (key must be normalized)", doc.TitlePageLines["draft date"])
	}
	if len(doc.Elements) != 2 {
		t.Fatalf("got %d elements, want 2", len(doc.Elements))
	}
	if doc.Elements[0].Line != 4 || doc.Elements[1].Line != 6 {
		t.Errorf("element lines = %d, %d; want 4, 6", doc.Elements[0].Line, doc.Elements[1].Line)
	}

	raw := ToJSON(Parse(input))
	var wire struct {
		TitlePageLines map[string]int `json:"titlePageLines"`
		Elements       []struct {
			Line int `json:"line"`
		} `json:"elements"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("wire JSON does not decode: %v", err)
	}
	if wire.TitlePageLines["title"] != 1 || len(wire.Elements) != 2 || wire.Elements[0].Line != 4 {
		t.Errorf("wire JSON lost lines: %+v", wire)
	}
}

// TestTypesFileDeclaresLineAndTitlePageLines mirrors the TitlePage guards for
// the new fields on Node and Document.
func TestTypesFileDeclaresLineAndTitlePageLines(t *testing.T) {
	raw, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatalf("read %s: %v", typesFile, err)
	}
	src := string(raw)

	nodeStart := strings.Index(src, "export interface Node {")
	if nodeStart < 0 {
		t.Fatalf("%s has no Node interface", typesFile)
	}
	nodeBody := src[nodeStart:]
	nodeBody = nodeBody[:strings.Index(nodeBody, "\n}")]
	if !regexp.MustCompile(`(?m)^\s*line:\s*number`).MatchString(nodeBody) {
		t.Errorf("Node in %s is missing 'line: number'", typesFile)
	}

	if !strings.Contains(src, "titlePageLines") {
		t.Errorf("%s is missing titlePageLines on Document", typesFile)
	}
}
