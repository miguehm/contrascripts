package parser

import (
	"testing"
)

func TestParseInlineEmpty(t *testing.T) {
	result := ParseInline("")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestParseInlinePlainText(t *testing.T) {
	result := ParseInline("hello world")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	pt, ok := result[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", result[0])
	}
	if pt.Text != "hello world" {
		t.Errorf("expected 'hello world', got %q", pt.Text)
	}
}

func TestParseInlineBold(t *testing.T) {
	result := ParseInline("**bold**")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	bold, ok := result[0].(Bold)
	if !ok {
		t.Fatalf("expected Bold, got %T", result[0])
	}
	if len(bold.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(bold.Content))
	}
	pt, ok := bold.Content[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText child, got %T", bold.Content[0])
	}
	if pt.Text != "bold" {
		t.Errorf("expected 'bold', got %q", pt.Text)
	}
}

func TestParseInlineItalic(t *testing.T) {
	result := ParseInline("_italic_")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	italic, ok := result[0].(Italic)
	if !ok {
		t.Fatalf("expected Italic, got %T", result[0])
	}
	if len(italic.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(italic.Content))
	}
	pt, ok := italic.Content[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText child, got %T", italic.Content[0])
	}
	if pt.Text != "italic" {
		t.Errorf("expected 'italic', got %q", pt.Text)
	}
}

func TestParseInlineBoldItalic(t *testing.T) {
	result := ParseInline("**_bold italic_**")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	bold, ok := result[0].(Bold)
	if !ok {
		t.Fatalf("expected Bold, got %T", result[0])
	}
	if len(bold.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(bold.Content))
	}
	italic, ok := bold.Content[0].(Italic)
	if !ok {
		t.Fatalf("expected Italic child, got %T", bold.Content[0])
	}
	if len(italic.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(italic.Content))
	}
	pt, ok := italic.Content[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText child, got %T", italic.Content[0])
	}
	if pt.Text != "bold italic" {
		t.Errorf("expected 'bold italic', got %q", pt.Text)
	}
}

func TestParseInlineMixed(t *testing.T) {
	result := ParseInline("hello _world_ **foo**")
	if len(result) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(result))
	}
	// PlainText "hello "
	pt0, ok := result[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", result[0])
	}
	if pt0.Text != "hello " {
		t.Errorf("expected 'hello ', got %q", pt0.Text)
	}
	// Italic "world"
	italic, ok := result[1].(Italic)
	if !ok {
		t.Fatalf("expected Italic, got %T", result[1])
	}
	if len(italic.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(italic.Content))
	}
	// PlainText " "
	pt2, ok := result[2].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", result[2])
	}
	if pt2.Text != " " {
		t.Errorf("expected ' ', got %q", pt2.Text)
	}
	// Bold "foo"
	bold, ok := result[3].(Bold)
	if !ok {
		t.Fatalf("expected Bold, got %T", result[3])
	}
	if len(bold.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(bold.Content))
	}
}

func TestParseInlineNoClosingBold(t *testing.T) {
	result := ParseInline("**no closing")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	pt, ok := result[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", result[0])
	}
	if pt.Text != "**no closing" {
		t.Errorf("expected '**no closing', got %q", pt.Text)
	}
}

func TestParseInlineNoClosingItalic(t *testing.T) {
	result := ParseInline("_no closing")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	pt, ok := result[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", result[0])
	}
	if pt.Text != "_no closing" {
		t.Errorf("expected '_no closing', got %q", pt.Text)
	}
}

func TestParseInlineMidWordUnderscore(t *testing.T) {
	result := ParseInline("foo_bar_baz")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	pt, ok := result[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", result[0])
	}
	if pt.Text != "foo_bar_baz" {
		t.Errorf("expected 'foo_bar_baz', got %q", pt.Text)
	}
}

func TestParseInlineAdjacentBold(t *testing.T) {
	result := ParseInline("**a****b**")
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	bold0, ok := result[0].(Bold)
	if !ok {
		t.Fatalf("expected Bold, got %T", result[0])
	}
	if len(bold0.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(bold0.Content))
	}
	bold1, ok := result[1].(Bold)
	if !ok {
		t.Fatalf("expected Bold, got %T", result[1])
	}
	if len(bold1.Content) != 1 {
		t.Fatalf("expected 1 child, got %d", len(bold1.Content))
	}
}

func TestParseInlinePlainTextBetweenMarkers(t *testing.T) {
	result := ParseInline("**a** plain **b**")
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	if _, ok := result[0].(Bold); !ok {
		t.Fatalf("expected Bold[0], got %T", result[0])
	}
	pt, ok := result[1].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText[1], got %T", result[1])
	}
	if pt.Text != " plain " {
		t.Errorf("expected ' plain ', got %q", pt.Text)
	}
	if _, ok := result[2].(Bold); !ok {
		t.Fatalf("expected Bold[2], got %T", result[2])
	}
}

func TestParseInlineDoubleUnderscore(t *testing.T) {
	result := ParseInline("__not italic__")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	pt, ok := result[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", result[0])
	}
	if pt.Text != "__not italic__" {
		t.Errorf("expected '__not italic__', got %q", pt.Text)
	}
}

func TestParseInlineNoClosingWithPrefix(t *testing.T) {
	result := ParseInline("hello **world")
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	pt0, ok := result[0].(PlainText)
	if !ok || pt0.Text != "hello " {
		t.Errorf("expected 'hello ', got %v", result[0])
	}
	pt1, ok := result[1].(PlainText)
	if !ok || pt1.Text != "**world" {
		t.Errorf("expected '**world', got %v", result[1])
	}

	resultItalic := ParseInline("hello _world")
	if len(resultItalic) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(resultItalic))
	}
	ptI0, ok := resultItalic[0].(PlainText)
	if !ok || ptI0.Text != "hello " {
		t.Errorf("expected 'hello ', got %v", resultItalic[0])
	}
	ptI1, ok := resultItalic[1].(PlainText)
	if !ok || ptI1.Text != "_world" {
		t.Errorf("expected '_world', got %v", resultItalic[1])
	}
}

func TestParseInlineItalicBoldNested(t *testing.T) {
	result := ParseInline("_**bold italic**_")
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	italic, ok := result[0].(Italic)
	if !ok {
		t.Fatalf("expected Italic, got %T", result[0])
	}
	if len(italic.Content) != 1 {
		t.Fatalf("expected 1 child in Italic, got %d", len(italic.Content))
	}
	bold, ok := italic.Content[0].(Bold)
	if !ok {
		t.Fatalf("expected Bold child, got %T", italic.Content[0])
	}
	if len(bold.Content) != 1 {
		t.Fatalf("expected 1 child in Bold, got %d", len(bold.Content))
	}
	pt, ok := bold.Content[0].(PlainText)
	if !ok || pt.Text != "bold italic" {
		t.Errorf("expected 'bold italic', got %v", bold.Content[0])
	}
}
