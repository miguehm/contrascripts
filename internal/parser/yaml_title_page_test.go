package parser

import "testing"

func TestParseYAMLFrontMatter(t *testing.T) {
	input := "title: Buscar al diablo\n" +
		"author:\n" +
		"  - Sarahi Zepeda Carrera\n" +
		"  - Miguel Hernandez Moreno\n" +
		"draft-date: 2026-08-13\n" +
		"contact: Casa\n" +
		"---\n" +
		"\n" +
		"EXT. CALLE - DIA\n" +
		"\n" +
		"Alguien corre.\n"
	f := Parse(input)

	if f.TitlePage.Title != "Buscar al diablo" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "Buscar al diablo")
	}
	if f.TitlePage.Author != "Sarahi Zepeda Carrera\nMiguel Hernandez Moreno" {
		t.Errorf("Author = %q, want multi-line authors", f.TitlePage.Author)
	}
	if f.TitlePage.DraftDate != "2026-08-13" {
		t.Errorf("DraftDate = %q, want %q", f.TitlePage.DraftDate, "2026-08-13")
	}
	if f.TitlePage.Contact != "Casa" {
		t.Errorf("Contact = %q, want %q", f.TitlePage.Contact, "Casa")
	}
	if len(f.Elements) == 0 {
		t.Fatal("expected body elements after front matter")
	}
}

func TestParseYAMLFrontMatterQuotedTitle(t *testing.T) {
	input := "title: \"La Casa de Papel\"\n" +
		"author: Alex Pina\n" +
		"---\n" +
		"\n" +
		"INT. BANCO - NIGHT\n"
	f := Parse(input)

	if f.TitlePage.Title != "La Casa de Papel" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "La Casa de Papel")
	}
	if f.TitlePage.Author != "Alex Pina" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "Alex Pina")
	}
}

func TestParseYAMLFrontMatterOpeningDelimiter(t *testing.T) {
	input := "---\n" +
		"title: Eclipse\n" +
		"author: Ana\n" +
		"---\n" +
		"\n" +
		"INT. AULA - DIA\n"
	f := Parse(input)

	if f.TitlePage.Title != "Eclipse" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "Eclipse")
	}
	if f.TitlePage.Author != "Ana" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "Ana")
	}
}

func TestParseNoFrontMatterStillParsesBody(t *testing.T) {
	// A header that is not Key: Value (e.g. bare author names) must NOT be
	// swallowed as a title page; it stays as body content.
	input := "Sarahi Zepeda Carrera\n" +
		"Miguel Hernandez Moreno\n" +
		"EXT. CALLE - DIA #1#\n"
	f := Parse(input)

	if f.TitlePage.Title != "" || f.TitlePage.Author != "" {
		t.Errorf("TitlePage should be empty, got %+v", f.TitlePage)
	}
	if len(f.Elements) == 0 {
		t.Fatal("expected body elements")
	}
}

func TestParseFrontMatterNoOpeningDelimiterWithContinuation(t *testing.T) {
	// Header without a leading "---" but with an indented plain-scalar
	// continuation (2-space indent) and a trailing "---" delimiter.
	input := "title: El último round\n" +
		"author:\n" +
		"  - Sara\n" +
		"contact:\n" +
		"  Moonlight Studios\n" +
		"  1234 Sunset Blvd\n" +
		"draft-date: 2026-08-13\n" +
		"---\n" +
		"EXT. CALLE - DIA\n"
	f := Parse(input)

	if f.TitlePage.Title != "El último round" {
		t.Errorf("Title = %q, want %q", f.TitlePage.Title, "El último round")
	}
	if f.TitlePage.Author != "Sara" {
		t.Errorf("Author = %q, want %q", f.TitlePage.Author, "Sara")
	}
	if f.TitlePage.Contact != "Moonlight Studios\n1234 Sunset Blvd" {
		t.Errorf("Contact = %q, want multiline value", f.TitlePage.Contact)
	}
	if f.TitlePage.DraftDate != "2026-08-13" {
		t.Errorf("DraftDate = %q, want %q", f.TitlePage.DraftDate, "2026-08-13")
	}
	if len(f.Elements) != 1 {
		t.Errorf("Elements len = %d, want 1", len(f.Elements))
	}
	if sc, ok := f.Elements[0].(SceneHeading); !ok || sc.Text != "EXT. CALLE - DIA" {
		t.Errorf("expected single scene heading, got %#v", f.Elements[0])
	}
}
