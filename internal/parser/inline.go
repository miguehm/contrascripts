package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseInline parses Fountain inline formatting markers (**bold**, _italic_)
// into a tree of InlineElement nodes.
func ParseInline(text string) []InlineElement {
	if text == "" {
		return nil
	}

	var result []InlineElement
	remaining := text

	for remaining != "" {
		// Find the next formatting marker
		boldIdx := strings.Index(remaining, "**")
		italicIdx := indexItalic(remaining)

		// No markers found — rest is plain text
		if boldIdx < 0 && italicIdx < 0 {
			result = append(result, PlainText{Text: remaining})
			break
		}

		// Pick the earliest marker
		idx := -1
		isBold := false
		if boldIdx >= 0 && (italicIdx < 0 || boldIdx <= italicIdx) {
			idx = boldIdx
			isBold = true
		} else {
			idx = italicIdx
			isBold = false
		}

		// Append any plain text before the marker
		if idx > 0 {
			result = append(result, PlainText{Text: remaining[:idx]})
		}

		if isBold {
			end := strings.Index(remaining[idx+2:], "**")
			if end < 0 {
				// No closing ** — treat remaining from idx as plain text
				result = append(result, PlainText{Text: remaining[idx:]})
				break
			}
			// Recurse into the content between ** markers
			inner := remaining[idx+2 : idx+2+end]
			bold := Bold{Content: ParseInline(inner)}
			result = append(result, bold)
			remaining = remaining[idx+2+end+2:]
		} else {
			// Find closing _ (must not be at start of word for opening marker)
			end := findClosingItalic(remaining, idx)
			if end < 0 {
				// No closing _ — treat remaining from idx as plain text
				result = append(result, PlainText{Text: remaining[idx:]})
				break
			}
			inner := remaining[idx+1 : end]
			italic := Italic{Content: ParseInline(inner)}
			result = append(result, italic)
			remaining = remaining[end+1:]
		}
	}

	return result
}

// indexItalic finds the next potential italic marker _.
// It only matches _ that opens emphasis (not mid-word, not doubled).
func indexItalic(text string) int {
	for i := 0; i < len(text); i++ {
		if text[i] == '_' {
			// Opening _: not preceded by alphanumeric
			if i > 0 {
				prevRune, _ := utf8.DecodeLastRuneInString(text[:i])
				if isAlnum(prevRune) {
					continue
				}
			}
			// Skip __ (double underscore is not italic in Fountain)
			if i+1 < len(text) && text[i+1] == '_' {
				i++ // skip both underscores
				continue
			}
			// Must have at least one char before closing
			if i+1 < len(text) && text[i+1] != ' ' {
				return i
			}
		}
	}
	return -1
}

// findClosingItalic finds the matching closing _ for an italic marker at openIdx.
func findClosingItalic(text string, openIdx int) int {
	for i := openIdx + 1; i < len(text); i++ {
		if text[i] == '_' {
			// Closing _: not followed by alphanumeric
			if i+1 < len(text) {
				nextRune, _ := utf8.DecodeRuneInString(text[i+1:])
				if isAlnum(nextRune) {
					continue
				}
			}
			// Skip __ (double underscore is not italic)
			if i+1 < len(text) && text[i+1] == '_' {
				i++ // skip both underscores
				continue
			}
			return i
		}
	}
	return -1
}

func isAlnum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
