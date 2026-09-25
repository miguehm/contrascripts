package parser

import (
	"reflect"
	"sort"
	"strconv"
	"unicode/utf8"
)

// appendJSON encodes v onto dst as JSON.
//
// This replaces encoding/json, which costs ~2.4 MB in a js/wasm binary
// (its static tables for every standard-library type) while reflect costs
// only ~24 KB. It deliberately supports just the subset of Go types the
// Document tree is built from: structs, slices, maps keyed by string,
// strings, integers and bools, honouring `json` struct tags. Any other kind
// encodes as null rather than panicking.
func appendJSON(dst []byte, v any) []byte {
	return appendJSONValue(dst, reflect.ValueOf(v))
}

func appendJSONValue(dst []byte, rv reflect.Value) []byte {
	if !rv.IsValid() {
		return append(dst, "null"...)
	}

	switch rv.Kind() {
	case reflect.String:
		return appendJSONString(dst, rv.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(dst, rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.AppendUint(dst, rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.AppendFloat(dst, rv.Float(), 'g', -1, 64)
	case reflect.Bool:
		return strconv.AppendBool(dst, rv.Bool())
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return append(dst, "null"...)
		}
		return appendJSONValue(dst, rv.Elem())
	case reflect.Slice:
		if rv.IsNil() {
			return append(dst, "null"...)
		}
		return appendJSONArray(dst, rv)
	case reflect.Map:
		return appendJSONMap(dst, rv)
	case reflect.Struct:
		return appendJSONStruct(dst, rv)
	}
	return append(dst, "null"...)
}

func appendJSONStruct(dst []byte, rv reflect.Value) []byte {
	rt := rv.Type()
	dst = append(dst, '{')

	first := true
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Tag.Get("json") == "-" {
			continue
		}
		fv := rv.Field(i)
		if field.Tag.Get("json") != "" && hasOption(field.Tag.Get("json"), "omitempty") && fv.IsZero() {
			continue
		}
		if !first {
			dst = append(dst, ',')
		}
		first = false
		dst = appendJSONString(dst, jsonTagName(field))
		dst = append(dst, ':')
		dst = appendJSONValue(dst, fv)
	}

	return append(dst, '}')
}

func appendJSONArray(dst []byte, rv reflect.Value) []byte {
	dst = append(dst, '[')
	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendJSONValue(dst, rv.Index(i))
	}
	return append(dst, ']')
}

func appendJSONMap(dst []byte, rv reflect.Value) []byte {
	if rv.IsNil() {
		return append(dst, "null"...)
	}

	keys := make([]string, 0, rv.Len())
	for _, k := range rv.MapKeys() {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)

	dst = append(dst, '{')
	for i, k := range keys {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendJSONString(dst, k)
		dst = append(dst, ':')
		dst = appendJSONValue(dst, rv.MapIndex(reflect.ValueOf(k)))
	}
	return append(dst, '}')
}

// jsonTagName returns the wire name of a field: the part of its json tag
// before any comma, or the Go field name when the tag is absent.
func jsonTagName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	if comma := indexByte(tag, ','); comma >= 0 {
		return tag[:comma]
	}
	return tag
}

func hasOption(tag, option string) bool {
	for tag != "" {
		var part string
		if comma := indexByte(tag, ','); comma >= 0 {
			part, tag = tag[:comma], tag[comma+1:]
		} else {
			part, tag = tag, ""
		}
		if part == option {
			return true
		}
	}
	return false
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

const jsonHex = "0123456789abcdef"

// appendJSONString writes s as a quoted JSON string. Invalid UTF-8 is replaced
// with U+FFFD so the output always parses, whatever bytes reached the parser.
func appendJSONString(dst []byte, s string) []byte {
	dst = append(dst, '"')

	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			switch c {
			case '"':
				dst = append(dst, '\\', '"')
			case '\\':
				dst = append(dst, '\\', '\\')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			default:
				if c < 0x20 {
					dst = append(dst, '\\', 'u', '0', '0', jsonHex[c>>4], jsonHex[c&0xF])
				} else {
					dst = append(dst, c)
				}
			}
			i++
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, `\ufffd`...)
			i++
			continue
		}
		dst = append(dst, s[i:i+size]...)
		i += size
	}

	return append(dst, '"')
}
