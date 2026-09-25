//go:build wasm

// Command wasm/pdf renders a Fountain script to a PDF entirely in the
// browser. It bundles the gofpdf renderer and the four embedded Courier
// Prime faces, so it is considerably larger than wasm/parser and should be
// loaded on demand rather than on first paint.
//
// Exported JS API:
//
//	renderPDF(text) -> Uint8Array
package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/user/fountain/internal/parser"
	"github.com/user/fountain/internal/renderer"
)

func main() {
	js.Global().Set("fountainRenderPDF", js.FuncOf(renderPDF))
	select {}
}

func renderPDF(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return throw("renderPDF: missing text argument")
	}

	var buf bytes.Buffer
	if err := renderer.New(parser.Parse(args[0].String())).RenderToWriter(&buf); err != nil {
		return throw("renderPDF: %v", err)
	}

	out := js.Global().Get("Uint8Array").New(len(buf.Bytes()))
	js.CopyBytesToJS(out, buf.Bytes())
	return out
}

func throw(format string, args ...any) any {
	js.Global().Get("console").Call("error", fmt.Sprintf(format, args...))
	return nil
}
