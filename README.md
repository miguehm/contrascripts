# fountain — `.fountain` → Screenplay PDF

A small Go CLI that parses screenplays in [Fountain](https://fountain.io) format and renders a production-style screenplay PDF with a professional title page.

- Fountain v1.1 parser: scene headings, action, character, dialogue, parentheticals, transitions, centered text, lyrics, sections, synopses, notes, boneyards, page breaks, forced elements
- Inline formatting: `**bold**`, `*italic*` and `_italic_`, including nested combinations and Unicode-safe wrapping
- Screenplay-accurate PDF: Courier 12pt, US Letter, correct margins, page numbers from page 2
- Opt-in scene numbering in both page margins (`scene-numbers: true`, off by default)
- Professional title page with 3-zone layout plus optional page 2 for synopsis / dedication / disclaimer
- Flexible metadata: classic `Key: value` title page or YAML frontmatter (`---`), with normalized keys, lists, and custom fields
- Zero-config CLI with stdin support

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
├── Makefile              # build / check / wasm / sizes
└── README.md
```

Pipeline: `Lexer.Tokenize` → `Parser.Parse` / `ParseFile` → `ParseInline` → `renderer.New(doc).RenderToFile/RenderToWriter`.

The same `internal/parser` and `internal/renderer` packages back all three
commands. `internal/` keeps them shareable inside the module without exposing
them as public API.

## WebAssembly

```bash
make wasm     # -> dist/fountain-parser.wasm, dist/fountain-pdf.wasm
make sizes    # raw / gzip / brotli footprint of both
```

`fountain-parser` bundles only the parser and linter, so it is small enough to
load on first paint. `fountain-pdf` adds gofpdf and the four embedded Courier
Prime faces, and is heavier, so it is meant to be loaded on demand.

| Module | raw | gzip | brotli |
|---|---|---|---|
| `fountain-parser` | 2 880 336 | 822 873 | 629 722 |
| `fountain-pdf` | 6 773 655 | 1 900 740 | 1 391 917 |

The floor for any `js/wasm` binary here is ~1.97 MB, which is the Go runtime
plus `syscall/js` alone. `internal/parser` uses a small reflect-based JSON
encoder (`jsonenc.go`) rather than `encoding/json`, which would add ~2.4 MB.

## Testing

```bash
make check    # gofmt, native + wasm go vet, go test
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

`testdata/` fixtures are versioned, so a missing one fails the test rather
than skipping it: an earlier `t.Skipf` there silently dropped all CLI coverage
when the package moved and stopped resolving the module root.

Note that `cmd/wasm-*` carry a `//go:build wasm` tag, so a plain `go vet ./...`
on Linux skips them. `make check` runs a `GOOS=js GOARCH=wasm` pass to keep
them type-checked.

## License

MIT © 2026 Contracontrol. See [LICENSE](LICENSE).
