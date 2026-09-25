# fountain — `.fountain` → Screenplay PDF

A small Go implementation that parses screenplays in [Fountain](https://fountain.io) format and renders a production-style screenplay PDF with a professional title page. It ships three front ends over the same `internal/` packages: a CLI, and two `js/wasm` modules for the browser (parse + lint, and PDF export).

- Fountain v1.1 parser: scene headings, action, character, dialogue, parentheticals, transitions, centered text, lyrics, sections, synopses, notes, boneyards, page breaks, forced elements
- Inline formatting: `**bold**`, `*italic*` and `_italic_`, including nested combinations and Unicode-safe wrapping
- Screenplay-accurate PDF: Courier 12pt, US Letter, correct margins, page numbers from page 2
- Opt-in scene numbering in both page margins (`scene-numbers: true`, off by default)
- Professional title page with 3-zone layout plus optional page 2 for synopsis / dedication / disclaimer
- Flexible metadata: classic `Key: value` title page or YAML frontmatter (`---`), with normalized keys, lists, and custom fields
- Zero-config CLI with stdin support
- Browser modules with a JS wrapper, TypeScript declarations, a live preview page and a real-browser smoke test — see [WebAssembly](#webassembly)

## Requirements

- Go 1.27+

Dependencies (from `go.mod`):

- `github.com/jung-kurt/gofpdf` — PDF generation

## Install

```bash
git clone <repo-url>
cd fountain
go mod download
make build            # -> bin/fountain
./bin/fountain -h
```

Or run without installing:

```bash
go run ./cmd/fountain -h
```

## Quick Start

1. Create `script.fountain`:

```fountain
Title: LA SEÑAL
Credit: Written by
Author: Jane Doe

EXT. SPACE - NIGHT

Stars fill the void.

ELENA
(into radio)
Can you hear me?
```

2. Convert to PDF:

```bash
fountain script.fountain              # -> script.pdf
fountain -o out.pdf script.fountain   # -> out.pdf
fountain script.fountain out.pdf      # -> out.pdf
cat script.fountain | fountain - -o out.pdf
```

3. Check available fixtures:

```bash
fountain testdata/minimal.fountain
fountain testdata/all-elements.fountain
fountain testdata/screenplay-complete.fountain
fountain testdata/extended-frontmatter.fountain
```

## CLI Usage

```text
Usage: fountain [options] <input.fountain> [output.pdf]

Options:
  -h	show help
  -help
    	show help
  -lint
    	print format warnings to stderr
  -o string
    	output PDF file path
  -v	show version
  -version
    	show version

Examples:
  fountain script.fountain              # -> script.pdf
  fountain -o out.pdf script.fountain    # -> out.pdf
  fountain script.fountain -o out.pdf    # -> out.pdf
  fountain -                            # read from stdin
```

Behavior:

- Flags may appear before, between or after the positional arguments, in any
  combination. `fountain script.fountain -o out.pdf` and
  `fountain -o out.pdf script.fountain` are equivalent.
- If `-o` is omitted and a second positional arg is given, it is used as output.
- If no output is given, `<input>.pdf` is used (`script.fountain` → `script.pdf`).
- Use `-` as input to read from stdin; default output is `output.pdf`.
- Use `--` to end flag parsing, for an input file whose name starts with `-`.
- `-h`/`-v` print to stdout; usage shown after a parse error goes to stderr.
- Exit codes: `0` = ok, `1` = error (missing input, parse/read failure, render failure), `2` = unknown flag.
- Warnings never change the exit code: with `--lint` the PDF is still generated and exit stays `0`.

Examples:

```bash
fountain script.fountain -o script.pdf
fountain -o out.pdf script.fountain
fountain -- -weird-name.fountain
fountain - -o out.pdf < script.fountain
fountain --lint script.fountain
fountain --help
fountain --version
```

## Lint (`--lint`)

Fountain is permissive by design: ambiguous lines silently fall back to `Action`
(or `Character` for all-caps lines after a blank line). `--lint` prints
diagnostics to `stderr` without changing the PDF:

```bash
$ fountain --lint script.fountain
warning: line 3 [UNCLOSED_BOLD]: odd number of ** markers, bold may not be closed
warning: line 5 [POSSIBLE_TRANSITION]: character cue looks like a transition missing colon (TO:)
```

| Code | Meaning |
|---|---|
| `UNCLOSED_BONEYARD` / `UNCLOSED_NOTE` / `UNCLOSED_FRONTMATTER` | `/*`, `[[` or `---` never closed |
| `UNCLOSED_BOLD` / `UNCLOSED_ITALIC` | `**`, `*` or `_` marker without closing pair (renders literally) |
| `POSSIBLE_CHARACTER` | All-caps line parsed as action; add a blank line before it for character, or `!` to force action |
| `POSSIBLE_TRANSITION` | Line ends with `TO` (missing `:`) or with `TO:` but not uppercase |
| `POSSIBLE_SCENE` | Line starts with `INT/EXT/EST` but is not a scene heading |
| `ORPHAN_DIALOGUE` / `ORPHAN_PAREN` | Dialogue or `(parenthetical)` with no character cue before it |
| `DIALOG_BEFORE_SCENE` | Character or dialogue before the first scene heading |
| `CHARACTER_NO_DIALOGUE` | Character cue with no dialogue after it |
| `DUAL_MISMATCH` | `^` must be the last character of the second cue |
| `SCENE_EMPTY` | Heading with no content before the next heading or EOF |
| `SCENE_NO_TIME` | Heading without ` - TIME` (e.g. `- DAY`); forced `.` headings are exempt |
| `TRANSITION_CHAIN` | Two transitions in a row |
| `BAD_EXTENSION` | Character `(extension)` unclosed or suspiciously long |
| `BAD_SECTION` / `BAD_SYNOPSIS` / `BAD_PAGEBREAK` | `#`, `=` or `===` with wrong spacing/content |
| `EMPTY_FORCED` | `@`, `!`, `>`, `~` or `>...<` with no text |
| `TITLE_MISSING` / `AUTHOR_MISSING` | Title page without `Title:` / `Author:` |
| `BAD_REVISION_COLOR` / `BAD_SCENE_NUMBERS` | Value outside the accepted set |
| `LONG_ACTION` / `LONG_DIALOGUE` / `LONG_PAREN` | Block longer than convention (>4 lines action/dialogue, >60 chars paren) |
| `OVERUSE_CENTERED` / `OVERUSE_LYRIC` / `OVERUSE_PAGEBREAK` / `OVERUSE_TRANSITION` | Element used far above normal frequency |
| `NAME_INCONSISTENT` | Near-duplicate character names (possible typo); extensions (`V.O.`) and dual `^` are ignored |

## Supported Fountain Syntax

| Element | Syntax | Example |
|---|---|---|
| Scene Heading | `INT./EXT.`, `.` prefix | `EXT. BRICK'S PATIO - DAY` / `.SNIPER SCOPE POV` |
| Action | Plain text fallback | `A gorgeous day. The sun is shining.` |
| Character | UPPERCASE after blank line, `@` forced | `STEEL` / `@McCLANE` / `BRICK ^` (dual) |
| Dialogue | Line(s) after Character / Parenthetical | `Beer's ready!` |
| Parenthetical | `(...)` after Character | `(beer raised)` / `(pause)` |
| Transition | `TO:`, `>`, forced | `CUT TO:` / `SMASH CUT TO:` / `> THE END <` is centered, not transition |
| Centered | `>text<` | `>BRICK & STEEL<` |
| Lyric | `~` prefix | `~Willy Wonka! Willy Wonka!` |
| Section | `#`, `##`, `###` | `# Act One` |
| Synopsis | `= ` prefix | `= Set up the characters.` |
| Note | `[[...]]`, multi-line | `[[It was supposed to be Vietnamese, right?]]` |
| Boneyard | `/* ... */`, multi-line | `/* This scene is commented out. */` |
| Page Break | `===` (line of 3+ `=`) | `===` |
| Forced Action | `!` prefix | `!SCANNING THE AISLES...` |

Notes:

- Sections and synopses are parsed but omitted from the PDF (Fountain v1.1 behavior).
- Notes and boneyards are parsed but invisible in the PDF.
- Multi-line dialogue, notes, and boneyards are supported.
- Screenplay formatting is intentional: scene headings and character names render in regular Courier, not bold.

Inline formatting:

```fountain
Steel's face FILLS the *Leupold Mark 4* scope.
_Permanently_.
**bold _nested italic_ text**
***bold italic***
```

Rules: `**bold**`, `*italic*`, `_italic_`, `***bold italic***`, nesting supported. Unclosed markers render literally. `__` is not italic. This project treats `_` and `*` as equivalent italics (Fountain spec reserves `_` for underline).

## Title Page and Frontmatter

Two equivalent ways to define metadata. If both exist, YAML frontmatter (`---`) takes precedence.

### 1. Classic Fountain title page

Key-value lines at the top of the file, before the first blank line:

```fountain
Title: BRICK & STEEL
Credit: Written by
Author: Stu Maschwitz
Draft date: 1/20/2012
Contact:
    Next Level Productions
    1588 Mission Dr.
```

Continuation lines must be indented.

### 2. YAML frontmatter

Optional `---` block at the top of the file. More expressive and Obsidian-friendly:

```yaml
---
title: "**LA _SEÑAL_**"
credit: Written by
author:
  - Sarahi Zepeda Carrera
  - Miguel Hernandez Moreno
draft-date: 2026-09-03
contact: Moonlight Studios
---
```

Rules:

- Keys are case-insensitive and normalized: `kebab-case` = `snake_case` = `with spaces` (`revision-color` = `revision_color` = `Revision Color`).
- Lists with `- ` are joined with `\n`.
- Unknown keys go to `Custom` and are also rendered on the title page.
- Only `title` supports inline formatting: `**bold**`, `*italic*` and `_italic_`, including nested combinations. It renders centered and uppercased; unclosed markers render literally and `__` is not italic. All other title-page fields render as plain text.

Supported keys and title-page zones:

| Keys | Title page zone |
|---|---|
| `title`, `credit`, `author`, `adapted-by`, `based-on`, `episode-title` | Top, centered |
| `logline`, `tagline`, `genre`, `subgenre`, `tone`, `format`, `season`, `episode`, `runtime`, `language`, `comps`, `themes`, `status`, `production-company`, `producer`, `director-attached`, `budget-level`, `source`, `revision` | Middle, centered |
| `draft-type`, `version`, `revision-color`, `revision-date`, `draft-date`, `pages`, `contact`, `address`, `email`, `phone`, `agent`, `manager`, `copyright`, `wga-registration`, `rights` + custom fields | Bottom (contact block) |
| `synopsis`, `dedication`, `disclaimer` | Page 2 |
| `scene-numbers` | Render setting, not printed on the title page |

`revision-color` whitelist: `white | blue | pink | yellow | green | goldenrod`.

### Scene numbers

Disabled by default. Set `scene-numbers: true` in the header to number scenes sequentially (`1`, `2`, …) in the left and right page margins, at the same height as each heading. The heading text itself is untouched. Applies to natural (`INT./EXT.`) and forced (`.`) headings. Accepts `true/yes/y/on/1`; anything else (including absent) means off.

```yaml
---
title: LA SEÑAL
scene-numbers: true
---

INT. CASA - NOCHE

 Acción.
```

Full example — see `testdata/extended-frontmatter.fountain`:

```yaml
---
title: LA SEÑAL
credit: Written by
author:
  - Sarahi Zepeda Carrera
  - Miguel Hernandez Moreno
draft-date: 2026-09-03
contact: Moonlight Studios
source: Original story
revision: Draft 3
logline: An operator detects an impossible signal and must decide whether to answer.
tagline: Silence speaks too.
synopsis: Elena picks up a signal that should not exist and risks everything to answer.
genre: Sci-fi
subgenre: Thriller
tone: Melancholic
themes:
  - memory
  - first contact
comps:
  - Arrival
  - Contact
format: series
season: 1
episode: 2
episode-title: The lost frequency
runtime: 52 min
pages: 58
language: es-MX
status: First draft
production-company: Moonlight Studios
producer: Ana Ruiz
director-attached: L. Mendez
budget-level: Low
adapted-by: Adapted by J. Doe
based-on: Original idea by S. Zepeda
copyright: © 2026 Moonlight Studios
wga-registration: WGA 123456
rights: All rights reserved.
dedication: For those who watch the sky.
disclaimer: Work of fiction.
draft-type: First draft
version: v1.0
revision-color: blue
revision-date: 2026-09-03
email: contact@moonlight.example
phone: +52 55 1234 5678
agent: North Agency
manager: M. Soto
address:
  1234 Sunset Blvd
  Los Angeles, CA 90028
my-custom-field: goes to Custom and is rendered too
---

EXT. SPACE - NIGHT

Stars fill the void.
```

## PDF Output Spec

- Paper: US Letter (8.5" × 11")
- Font: Courier 12pt (embedded Courier Prime when available, fallback to core Courier)
- Margins: 1.5" left, 1.0" top/right/bottom (6.0" usable width)
- Layout: Character 3.7", Dialogue 2.5"–6.0", Parenthetical 3.1"–5.0", Transition flush right, Action / Scene full width
- Page numbers top-right from page 2 (title page unnumbered)
- Scene numbers (when enabled): plain number in both margins, heading text unchanged

## Project Structure

```text
.
├── cmd/
│   ├── fountain/         # CLI entry point (-o/-h/-v, stdin, output resolution)
│   ├── wasm-parser/      # js/wasm: parse + lint for the browser
│   └── wasm-pdf/         # js/wasm: PDF export for the browser
├── internal/
│   ├── parser/
│   │   ├── types.go      # AST: Fountain, TitlePage, Element + Inline nodes
│   │   ├── lexer.go      # Line-by-line tokenizer + context rules
│   │   ├── parser.go     # Tokens -> AST, title page, multi-line handling
│   │   ├── inline.go     # **bold** / *italic* / _italic_ parser
│   │   ├── lint.go       # format warnings (--lint)
│   │   ├── json.go       # AST -> JSON wire format for non-Go renderers
│   │   ├── jsonenc.go    # minimal reflect-based JSON encoder
│   │   └── *_test.go     # Parser, lexer, inline, lint, JSON tests
│   └── renderer/
│       ├── pdf.go        # gofpdf renderer, title page + screenplay layout
│       ├── fonts.go      # Courier Prime embedding
│       └── pdf_test.go   # Structural PDF tests
├── testdata/             # .fountain fixtures
├── web/
│   ├── fountain.mjs      # JS wrapper: boots wasm_exec.js, lazy-loads PDF
│   ├── fountain.d.ts     # its types, for a TypeScript frontend
│   ├── dist-package.json # template for dist/package.json (version injected)
│   ├── index.html        # live preview page, no build step
│   ├── serve.mjs         # static server for the smoke test
│   ├── tests/            # Playwright specs: the modules in Chromium, the
│   │                     # packaging contract in Node
│   └── vite/             # React + Vite consumer, run in dev and in build
├── Makefile              # build / check / wasm / dist / package / sizes / smoke
└── README.md
```

Pipeline: `Lexer.Tokenize` → `Parser.Parse` / `ParseFile` → `ParseInline` → `renderer.New(doc).RenderToFile/RenderToWriter`.

The same `internal/parser` and `internal/renderer` packages back all three
commands. `internal/` keeps them shareable inside the module without exposing
them as public API.

## WebAssembly

```bash
make dist     # -> dist/ (the folder a web project consumes)
make package DIR=public/fountain   # stage it into another project
make smoke    # build, then run both modules in a real browser
make sizes    # raw / gzip / brotli footprint of both modules
```

`make dist` is the target that matters. `make wasm` on its own emits two `.wasm`
files that **no browser can instantiate**: a `js/wasm` binary is not a library,
it is a standalone module that needs Go's runtime shim and registers its API as
globals on `globalThis`. `dist/` is the complete set:

| File | What it is |
|---|---|
| `fountain-parser.wasm` | parser + linter, no fonts, no PDF engine |
| `fountain-pdf.wasm` | gofpdf + four embedded Courier Prime faces |
| `wasm_exec.js` | Go's runtime shim, copied out of `GOROOT` — **required** |
| `fountain.mjs` | JS wrapper over the globals |
| `fountain.d.ts` | the wrapper's types, for a frontend that cannot read JSDoc |
| `package.json` | makes the folder a package: `exports`, `types`, the CLI's version |
| `manifest.json` | which Go produced the modules, and their sha256 |

`wasm_exec.js` is never committed: it must be the one shipped with the toolchain
that compiled the `.wasm`, and a mismatched copy fails at `go.run()` with an
error that does not mention the cause. `manifest.json` records the pairing, and
`make smoke` fails if it ever drifts.

### Using it

`fountain.mjs` resolves its `.wasm` siblings from its own URL, so a consumer
copies the whole folder and needs no arguments:

```js
import { createFountain } from './fountain.mjs'

const fountain = await createFountain()

fountain.parse(text)              // -> Document   (sync)
fountain.lint(text)               // -> Warning[]  (sync)
await fountain.renderPDF(text)    // -> Uint8Array (async, lazy 6.7 MB load)
```

Both `parse` and `lint` are synchronous: the Go globals return immediately.
`renderPDF` is the only asynchronous call, because it fetches its module on
first use. The wrapper also:

- parses the JSON the globals return, so you get objects rather than strings
- throws `FountainError` where the raw globals return `null` and log to the
  console, so failures are catchable
- falls back from `instantiateStreaming` when the server sends the wrong
  `Content-Type` for `.wasm`

`web/index.html` is a working live preview — two panes, no build step, no
framework — and is the intended shape of a consumer.

### Using it from a frontend

`dist/` is not committed, so a frontend gets it from the build:

```bash
make -C ../fountain package DIR=public/fountain
```

That stages the four runtime assets — the two `.wasm` files, `wasm_exec.js` and
the manifest — into your `public/` folder, where a bundler serves them
untouched. It deliberately does **not** stage `fountain.mjs`: a second copy
sitting in `public/` would be a file the app never imports, and the Vite
integration test asserts it is not served. `web/vite/` is a working React +
Vite consumer wired exactly this way, and it is exercised by `make smoke` in
both `dev` and `build`.

**Keep the wrapper in source and the heavy files in `public/`.** A bundler
compiles your source, so `fountain.mjs` ends up inside a chunk whose
`import.meta.url` is no longer a sibling of the `.wasm` files. It therefore
needs to be told where they are:

```js
import { createFountain } from '../../../fountain/dist/fountain.mjs' // or your path

const fountain = await createFountain({ base: '/fountain' })
```

`base` is a URL, resolved against the page, so a site-absolute path is fine.
Without it the wrapper falls back to its own location, which is correct only
when nothing rewrote it — that is, when you serve it unbundled.

A Vite app cannot import a `.mjs` out of `public/` at all: it rejects the
import, and the `?url` escape hatch it suggests in the error answers with the
SPA `index.html` and a `200`, so the module silently receives HTML. Keeping the
wrapper in source sidesteps the whole thing. Vite also prints a warning that
`new URL('.', import.meta.url) doesn't exist at build time` — that one is
expected here, it is the wrapper resolving itself at runtime.

**TypeScript.** `fountain.d.ts` sits next to the wrapper in `dist/`, so
`tsc` picks it up from the same path you import the `.mjs` by, or through a
`paths` entry if you copied it into your own tree:

```jsonc
// tsconfig.json
{
  "compilerOptions": {
    "paths": { "fountain": ["./fountain/dist/fountain.mjs"] }
  }
}
```

It is hand-written, because nothing in a browser project can compile a Go
struct into a `.d.ts`. Two tests keep it honest: `internal/parser` asserts every
`TitlePage` field on the wire and every element type is declared, and
`web/tests/package.spec.mjs` asserts every typedef in the wrapper has a
declaration and that `dist/` ships a current copy of both files.

**It is a browser module.** `createFountain()` injects a `<script>` for the
shim and `fetch`es the `.wasm`, so it needs a `document`. Called during an SSR
pass, a prerender or in plain Node it rejects with a `FountainError` that says
exactly that, instead of failing later as a `fetch` against a `file://` URL or
as a 404 on the shim. Load it from client code:

```js
// React: a dynamic import inside the effect, not a top-level import
useEffect(() => {
  let alive = true
  import('../../fountain/dist/fountain.mjs').then(async (m) => {
    const f = await m.createFountain({ base: '/fountain' })
    if (alive) setFountain(f)
  })
  return () => {
    alive = false
  }
}, [])
```

```jsx
// Next.js: keep the whole editor out of the server render
'use client'
import dynamic from 'next/dynamic'
const Editor = dynamic(() => import('./Editor'), { ssr: false })
```

**Boot once and share.** `createFountain()` is async and starts a Go runtime,
so calling it in an effect means one runtime per mount — and React StrictMode
double-invokes effects in development, while Vite re-evaluates modules on HMR.
A module-level variable is not enough, because HMR resets it:

```js
const INSTANCE = Symbol.for('fountain.instance')

export function loadFountain() {
  globalThis[INSTANCE] ??= boot()
  return globalThis[INSTANCE]
}
```

`Symbol.for` is the same symbol across realms and survives module re-evaluation.
`web/vite/src/fountain.js` is the full version, and its smoke test asserts the
runtime count stays at 1. Note what that ceiling means: there is no `dispose()`,
so the heap lives as long as the page. One instance per document, and no way to
free it early.

**`parse` and `lint` are synchronous** and block the main thread. On a long
script that shows up as jank while typing; collapsing keystrokes into one call
per animation frame is enough. The same caveat applies to the globals directly,
and it is why `renderPDF` is the only asynchronous call.

Serving `.wasm` as `application/wasm` lets the browser stream-compile it. The
wrapper falls back to a plain fetch when a server gets that wrong, so a
misconfigured CDN costs startup speed rather than correctness.

### Without a build step

`dist/` is also the folder to publish as-is: no bundler, no `base`, no import
map, because the wrapper finds its own `.wasm` siblings from its own URL. Copy
it to a static host and import it by path.

```html
<script type="module">
  import { createFountain } from '/fountain/fountain.mjs'

  const fountain = await createFountain()
  const doc = fountain.parse('Title: T\n\nEXT. CASA - DIA\n\nTexto.\n')
  document.querySelector('#out').textContent = JSON.stringify(doc, null, 2)
</script>
```

`web/index.html` is a fuller version of the same shape: two panes, no
framework, and the intended starting point.

Two host details are worth getting right. Neither breaks anything — the wrapper
falls back — but both cost startup speed.

| | |
|---|---|
| `.wasm` served as `application/wasm` | the browser stream-compiles it. Without the type, `instantiateStreaming` throws and the whole 2.9 MB is fetched before compiling |
| `script-src` includes `'wasm-unsafe-eval'` | a strict CSP blocks WebAssembly compilation outright. The shim itself needs no `unsafe-eval` — it uses no `eval` |

```nginx
# nginx: both of the above, and the binaries are immutable
types { application/wasm wasm; }
location /fountain/ { add_header Cache-Control "public, max-age=31536000, immutable"; }
```

```json
// Vercel: vercel.json — .wasm already gets the right content-type here
{
  "headers": [
    {
      "source": "/(.*)",
      "headers": [
        { "key": "Content-Security-Policy", "value": "script-src 'self' 'wasm-unsafe-eval'" }
      ]
    }
  ]
}
```

```text
# Netlify: _headers — same header, same file
/*
  Content-Security-Policy: script-src 'self' 'wasm-unsafe-eval'
```

Vercel and Netlify both serve `.wasm` as `application/wasm` already; the
nginx block is for Apache, a plain filesystem, or anything hand-rolled, where
`application/octet-stream` costs the whole download before compiling.

If the CSP also forbids injecting scripts, include the shim in the page and
tell the wrapper to use it:

```html
<script src="/fountain/wasm_exec.js"></script>
<script type="module">
  import { createFountain } from '/fountain/fountain.mjs'
  const fountain = await createFountain({ injectWasmExec: false })
</script>
```

### Calling the globals directly

If you would rather skip the wrapper, the three globals are available on
`globalThis` once each module has run. Each needs its own `Go` instance, and the
return values differ: `fountainParse` and `fountainLint` return **JSON strings**,
`fountainRenderPDF` returns a `Uint8Array`, and a failure is `null` plus a
`console.error` rather than a thrown exception.

```js
const go = new Go()
go.run(instance)                 // not awaited: main() ends in select{}
const doc = JSON.parse(globalThis.fountainParse(text))
```

### Two modules, two runtimes

Each compiled module is its own Go runtime with its own heap. They coexist in
one page — `make smoke` asserts it, loading the parser, then the PDF into the
same document, and checking the parser still answers afterwards — but they do
not share memory or a scheduler, so a session that exports pays for two heaps.

The split is what keeps `fountain-pdf` off the first paint, and that is the
whole point of it:

| Module | raw | gzip | brotli |
|---|---|---|---|
| `fountain-parser` | 2 880 388 | 822 908 | 628 948 |
| `fountain-pdf` | 6 773 655 | 1 900 750 | 1 392 381 |

Fusing them would put gofpdf and the embedded fonts on the critical path for
users who only ever type and preview. `make sizes` exists to keep that
trade-off honest: it is the number that says whether the split is still worth
keeping. The floor for any `js/wasm` binary here is ~1.97 MB, which is the Go
runtime plus `syscall/js` alone. `internal/parser` uses a small reflect-based
JSON encoder (`jsonenc.go`) rather than `encoding/json`, which would add ~2.4 MB.

### JSON wire format

`parse` returns a `Document`. `type` is the lowercase element name, matching
what the PDF draws — a forced `.` scene heading comes back as `sceneHeading`,
not as a `prefix`.

```jsonc
{
  "titlePage": { "title": "LA SEÑAL", "author": "Jane Doe", "custom": {} },
  "elements": [
    {
      "type": "sceneHeading",       // action, character, dialogue, parenthetical,
      "text": "EXT. SPACE - NIGHT", // transition, centered, lyric, section,
      "uppercase": true             // synopsis, pageBreak, note, boneyard
    },
    {
      "type": "action",
      "text": "A **loud** bang.",
      "baseItalic": true,
      "inline": [
        { "type": "text",  "text": "A " },
        { "type": "bold",  "content": [{ "type": "text", "text": "loud" }] },
        { "type": "text",  "text": " bang." }
      ]
    }
  ]
}
```

- `uppercase` and `baseItalic` are **hints**, not transformations: `text` is
  exactly as parsed, and a renderer is expected to apply them the way the PDF
  does.
- `inline` is present only for the element types where the PDF honours `**bold**`
  and `*italic*`.
- `titlePage` and `custom` always encode as an object, and a clean `lint`
  encodes as `[]`. Both are guaranteed rather than left to Go's nil encoding,
  because JavaScript consumers index and iterate them unguarded.

## Testing

```bash
make check    # gofmt, native + wasm go vet, go test
make smoke    # make dist, then the browser suites
go test ./...
```

Test strategy:

- Unit tests per element type (parser, lexer, inline, lint, JSON).
- Integration tests with real `.fountain` files in `testdata/`.
- Structural PDF tests (header, EOF, page objects) — no golden PDFs because `gofpdf` embeds non-deterministic timestamps/metadata.
- Exhaustive in-process tests of the flag/positional matrix (`TestParseArgs`),
  covering flags before, between and after positionals, `--` and `-` handling,
  and the long flag spellings.
- End-to-end CLI tests (`cmd/fountain/main_test.go`) that run a real binary.
  `TestMain` builds it once into a temp dir instead of shelling out to
  `go run` per case, so the suite is not dominated by repeated compilation.
  Cleanup is explicit rather than deferred, because `os.Exit` skips defers.
- A browser smoke test (`web/`, Playwright) for everything the Go compiler
  cannot see. It runs in Chromium because a `js/wasm` binary that compiles is
  not yet a binary that runs: the failure modes are a `wasm_exec.js` that no
  longer matches the toolchain, and the two modules fighting over `globalThis`.
  The spec is a single serial run on one page, loading the parser and then the
  PDF into the same document, because splitting it across pages would hide the
  failure it exists to catch. A second spec drives `web/index.html` end to end,
  so the usage this README documents is itself tested.
- A Vite integration fixture (`web/vite/`) run in **both** `vite dev` and
  `vite build && vite preview`, because those are different code paths and
  passing one is no evidence about the other — the wrapper only loses its
  `import.meta.url` in the built one. It asserts a single runtime survives
  StrictMode, that the assets are fetched from `public/` and not bundled, that
  the wrapper is *not* among them, and that no request 404s. Two of the bugs it
  exists to catch were in the wrapper rather than the bundler, both from `base`
  handling.
- A Node-side packaging spec (`web/tests/package.spec.mjs`) for the failures a
  browser cannot show: a file missing from `dist/`, a stale copy of the wrapper,
  a `package.json` pointing at a specifier that is not there, a version that
  disagrees with the CLI's, and `createFountain()` rejecting outside a browser
  with an error that names the cause. It also checks the `.d.ts` against the
  wrapper's JSDoc, which nothing else compiles.
- `internal/parser/json_types_test.go` covers the other half of the types
  contract, from the side that has the truth: every `TitlePage` field the wire
  format emits is declared in `fountain.d.ts`, nothing is declared that is not
  emitted, no field gained an `omitempty`, and every element type the parser can
  produce is in the `ElementType` union.

`testdata/` fixtures are versioned, so a missing one fails the test rather
than skipping it: an earlier `t.Skipf` there silently dropped all CLI coverage
when the package moved and stopped resolving the module root.

Note that `cmd/wasm-*` carry a `//go:build wasm` tag, so a plain `go vet ./...`
on Linux skips them. `make check` runs a `GOOS=js GOARCH=wasm` pass to keep
them type-checked.

## License

MIT © 2026 Contracontrol. See [LICENSE](LICENSE).
