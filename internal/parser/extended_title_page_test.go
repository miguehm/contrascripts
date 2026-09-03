package parser

import "testing"

func TestParseExtendedFrontMatter(t *testing.T) {
	f, err := ParseFile("../../testdata/extended-frontmatter.fountain")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}
	tp := f.TitlePage
	if tp.Logline == "" {
		t.Error("Logline empty")
	}
	if tp.Genre != "Ciencia ficción" {
		t.Errorf("Genre = %q", tp.Genre)
	}
	if tp.Comps == "" {
		t.Error("Comps empty")
	}
	if tp.Format != "feature" {
		t.Errorf("Format = %q", tp.Format)
	}
	if tp.Email != "contacto@moonlight.example" {
		t.Errorf("Email = %q", tp.Email)
	}
	if tp.RevisionColor != "white" {
		t.Errorf("RevisionColor = %q", tp.RevisionColor)
	}
	if tp.Copyright == "" || tp.Synopsis == "" {
		t.Error("expected Copyright and Synopsis")
	}
}

func TestParseExtendedKeysNormalization(t *testing.T) {
	input := "revision-color: blue\nrevision_date: 2026-09-03\ndirector-attached: X\nbased_on: Y\n---\n\nINT. X - DAY\n"
	f := Parse(input)
	if f.TitlePage.RevisionColor != "blue" {
		t.Errorf("RevisionColor = %q", f.TitlePage.RevisionColor)
	}
	if f.TitlePage.RevisionDate != "2026-09-03" {
		t.Errorf("RevisionDate = %q", f.TitlePage.RevisionDate)
	}
	if f.TitlePage.Director != "X" {
		t.Errorf("Director = %q", f.TitlePage.Director)
	}
	if f.TitlePage.BasedOn != "Y" {
		t.Errorf("BasedOn = %q", f.TitlePage.BasedOn)
	}
}
