package parser

import (
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
