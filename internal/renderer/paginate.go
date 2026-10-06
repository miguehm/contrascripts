package renderer

import (
	"encoding/json"

	"github.com/user/fountain/internal/parser"
)

// ScenePage maps a scene heading's 1-based source line to the absolute
// 1-based PDF page where the heading starts. Page numbers include the
// title page, matching the pdf.js numbering a web preview sees.
type ScenePage struct {
	Line int `json:"line"`
	Page int `json:"page"`
}

// Paginate lays out input and reports the page of every scene heading.
// A script without scenes yields an empty array, never null. It runs the
// full render to let the page-breaking rules decide, then discards the
// PDF bytes and keeps only the recorded pages.
func Paginate(input string) []byte {
	r := New(parser.Parse(input))
	r.render()
	pages := r.scenePages
	if pages == nil {
		pages = []ScenePage{}
	}
	out, err := json.Marshal(pages)
	if err != nil {
		return []byte("[]")
	}
	return out
}
