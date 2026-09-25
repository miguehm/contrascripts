//go:build wasm

// Command wasm/parser exposes the Fountain parser and linter to the browser
// over WebAssembly. It ships no PDF renderer, no fonts and no image codecs,
// so it stays small enough to load on first paint.
//
// Exported JS API:
//
//	parse(text) -> string   // JSON Document
//	lint(text)  -> string   // JSON Warning[]
package main

import (
	"fmt"
	"syscall/js"

	"github.com/user/fountain/internal/parser"
)

func main() {
	js.Global().Set("fountainParse", js.FuncOf(parse))
	js.Global().Set("fountainLint", js.FuncOf(lint))
	select {}
}

func parse(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return throw("parse: missing text argument")
	}
	return string(parser.ToJSON(parser.Parse(args[0].String())))
}

func lint(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return throw("lint: missing text argument")
	}
	return string(parser.ToWarningsJSON(args[0].String()))
}

func throw(format string, args ...any) any {
	js.Global().Get("console").Call("error", fmt.Sprintf(format, args...))
	return nil
}
