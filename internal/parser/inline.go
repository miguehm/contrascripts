package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseInline parses Fountain inline formatting markers (**bold**, *italic*, _italic_)
// into a tree of InlineElement nodes.
func ParseInline(text string) []InlineElement {
	if text == "" {
		return nil
	}

	var result []InlineElement
	remaining := text

	for remaining != "" {
		boldIdx := strings.Index(remaining, "**")
		italicIdx := indexItalic(remaining)
		starIdx := indexSingleItalic(remaining)

		if boldIdx < 0 && italicIdx < 0 && starIdx < 0 {
			result = append(result, PlainText{Text: remaining})
			break
		}

		idx := -1
		kind := 0
		if boldIdx >= 0 {
			idx = boldIdx
			kind = 1
		}
		if italicIdx >= 0 && (idx < 0 || italicIdx < idx) {
			idx = italicIdx
			kind = 2
		}
		if starIdx >= 0 && (idx < 0 || starIdx < idx) {
			idx = starIdx
			kind = 3
		}

		if idx > 0 {
			result = append(result, PlainText{Text: remaining[:idx]})
		}

		if kind == 1 {
			after := remaining[idx+2:]
			if strings.HasPrefix(after, "*") {
				if close := strings.LastIndex(after, "**"); close > 1 {
					inner := after[:close]
					bold := Bold{Content: ParseInline(inner)}
					result = append(result, bold)
					remaining = after[close+2:]
					continue
				}
			}
			end := strings.Index(after, "**")
			if end < 0 {
				result = append(result, PlainText{Text: remaining[idx:]})
				break
			}
			inner := remaining[idx+2 : idx+2+end]
			bold := Bold{Content: ParseInline(inner)}
			result = append(result, bold)
			remaining = remaining[idx+2+end+2:]
		} else if kind == 2 {
			end := findClosingItalic(remaining, idx)
			if end < 0 {
				result = append(result, PlainText{Text: remaining[idx:]})
				break
			}
			inner := remaining[idx+1 : end]
			italic := Italic{Content: ParseInline(inner)}
			result = append(result, italic)
			remaining = remaining[end+1:]
		} else {
			end := findClosingSingle(remaining, idx)
			if end < 0 {
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

func indexSingleItalic(text string) int {
	for i := 0; i < len(text); i++ {
		if text[i] == '*' {
			if i+1 < len(text) && text[i+1] == '*' {
				i++
				continue
			}
			if i > 0 && text[i-1] == '*' {
				continue
			}
			if i > 0 {
				prevRune, _ := utf8.DecodeLastRuneInString(text[:i])
				if isAlnum(prevRune) {
					continue
				}
			}
			if i+1 < len(text) && text[i+1] != ' ' {
				return i
			}
		}
	}
	return -1
}

func findClosingSingle(text string, openIdx int) int {
	for i := openIdx + 1; i < len(text); i++ {
		if text[i] == '*' {
			if i+1 < len(text) && text[i+1] == '*' {
				i++
				continue
			}
			if i > 0 && text[i-1] == '*' {
				continue
			}
			if i+1 < len(text) {
				nextRune, _ := utf8.DecodeRuneInString(text[i+1:])
				if isAlnum(nextRune) {
					continue
				}
			}
			return i
		}
	}
	return -1
}

func isAlnum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
