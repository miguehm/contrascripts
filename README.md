# fountain — `.fountain` → Screenplay PDF

A small Go CLI that parses screenplays in [Fountain](https://fountain.io) format and renders a production-style screenplay PDF with a professional title page.

- Fountain v1.1 parser: scene headings, action, character, dialogue, parentheticals, transitions, centered text, lyrics, sections, synopses, notes, boneyards, page breaks, forced elements
- Inline formatting: `**bold**` and `_italic_`, including nested combinations and Unicode-safe wrapping
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
go build -o fountain .
./fountain -h
```

Or run without installing:

```bash
go run . -h
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
  -o string
        output PDF file path
  -h            show help
  -v            show version (fountain v1.0.0)
```

Behavior:

- If `-o` is omitted and a second positional arg is given, it is used as output.
- If no output is given, `<input>.pdf` is used (`script.fountain` → `script.pdf`).
- Use `-` as input to read from stdin; default output is `output.pdf`.
- Exit codes: `0` = ok, `1` = error (missing input, parse/read failure, render failure).

Examples:

```bash
fountain script.fountain -o script.pdf
fountain -o out.pdf script.fountain
fountain - < script.fountain
fountain --help
fountain --version
```

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
```

Is not valid — use Fountain style:

```fountain
Steel's face FILLS the **Leupold Mark 4** scope.
_Permanently_.
**bold _nested italic_ text**
```

Rules: `**bold**`, `_italic_`, nesting supported. Unclosed markers render literally. `__` is not italic.

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
- Only `title` supports inline formatting: `**bold**` and `_italic_`, including nested combinations. It renders centered and uppercased; unclosed markers render literally and `__` is not italic. All other title-page fields render as plain text.

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
├── main.go                  # CLI entry point (-o/-h/-v, stdin, output resolution)
├── go.mod / go.sum
├── internal/
│   ├── parser/
│   │   ├── types.go         # AST: Fountain, TitlePage, Element + Inline nodes
│   │   ├── lexer.go         # Line-by-line tokenizer + context rules
│   │   ├── parser.go        # Tokens -> AST, title page, multi-line handling
│   │   ├── inline.go        # **bold** / _italic_ parser
│   │   └── *_test.go        # Parser, lexer, inline, title-page tests
│   └── renderer/
│       ├── pdf.go           # gofpdf renderer, title page + screenplay layout
│       ├── fonts.go         # Courier Prime embedding
│       └── pdf_test.go      # Structural PDF tests
├── testdata/
│   ├── minimal.fountain
│   ├── all-elements.fountain
│   ├── screenplay-complete.fountain
│   └── extended-frontmatter.fountain
└── README.md
```

Pipeline: `Lexer.Tokenize` → `Parser.Parse` / `ParseFile` → `ParseInline` → `renderer.New(doc).RenderToFile/RenderToWriter`.

## Testing

```bash
go test ./...
go vet ./...
gofmt -l .
```

Test strategy:

- Unit tests per element type (parser, lexer, inline).
- Integration tests with real `.fountain` files in `testdata/`.
- Structural PDF tests (header, EOF, page objects) — no golden PDFs because `gofpdf` embeds non-deterministic timestamps/metadata.
- End-to-end CLI tests (`main_test.go`).

## License

MIT © 2026 Contracontrol. See [LICENSE](LICENSE).
