package renderer

import (
	"encoding/json"
	"testing"
)

// The tests here cover Paginate, the scene line → absolute PDF page map the
// web preview uses for exact scene jumps. Pages are absolute like pdf.js
// sees them: without front matter, body page 1 is PDF page 1.

func decodePages(t *testing.T, src string) []ScenePage {
	t.Helper()
	var out []ScenePage
	if err := json.Unmarshal(Paginate(src), &out); err != nil {
		t.Fatalf("Paginate(%q) returned invalid JSON: %v", src, err)
	}
	return out
}

func TestPaginateTwoScenes(t *testing.T) {
	src := "EXT. PATIO - DÍA\n\nAcción.\n\nINT. CASA - NOCHE\n\nMás acción.\n"
	pages := decodePages(t, src)
	if len(pages) != 2 {
		t.Fatalf("Paginate returned %d scenes, want 2: %v", len(pages), pages)
	}
	if pages[0].Line != 1 || pages[0].Page != 1 {
		t.Errorf("first scene = %+v, want {Line:1 Page:1}", pages[0])
	}
	if pages[1].Line != 5 || pages[1].Page != 1 {
		t.Errorf("second scene = %+v, want {Line:5 Page:1}", pages[1])
	}
}

func TestPaginateSecondSceneOnNextPage(t *testing.T) {
	src := fillerParagraph(linesPerPage) + "\n\nINT. LEJOS - DÍA\n"
	pages := decodePages(t, src)
	if len(pages) != 1 {
		t.Fatalf("Paginate returned %d scenes, want 1: %v", len(pages), pages)
	}
	if pages[0].Line != linesPerPage+2 {
		t.Errorf("scene line = %d, want %d", pages[0].Line, linesPerPage+2)
	}
	if pages[0].Page != 2 {
		t.Errorf("scene page = %d, want 2", pages[0].Page)
	}
}

func TestPaginateDuplicateHeadings(t *testing.T) {
	src := "INT. CASA - DÍA\n\nAcción.\n\nINT. CASA - DÍA\n\nOtra.\n"
	pages := decodePages(t, src)
	if len(pages) != 2 {
		t.Fatalf("Paginate returned %d scenes, want 2: %v", len(pages), pages)
	}
	if pages[0].Line == pages[1].Line {
		t.Errorf("duplicate headings share line %d, want distinct lines", pages[0].Line)
	}
	if pages[0].Page != pages[1].Page {
		t.Errorf("duplicate headings on pages %d and %d, want the same page",
			pages[0].Page, pages[1].Page)
	}
}

func TestPaginateForcedHeading(t *testing.T) {
	src := ".INT. CASA - DÍA\n\nAcción.\n"
	pages := decodePages(t, src)
	if len(pages) != 1 {
		t.Fatalf("Paginate returned %d scenes, want 1: %v", len(pages), pages)
	}
	if pages[0].Line != 1 || pages[0].Page != 1 {
		t.Errorf("forced scene = %+v, want {Line:1 Page:1}", pages[0])
	}
}

func TestPaginateWithoutScenes(t *testing.T) {
	got := string(Paginate("Una acción.\n"))
	if got != "[]" {
		t.Errorf("Paginate without scenes = %q, want %q (empty array, never null)", got, "[]")
	}
}
