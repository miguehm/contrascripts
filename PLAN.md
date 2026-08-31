# Plan: `fountain` — CLI Go para convertir `.fountain` → PDF

## Arquitectura

```
fountain/
├── main.go                  # CLI entry point
├── go.mod
├── internal/
│   ├── parser/
│   │   ├── parser.go        # Core Fountain v1.1 parser
│   │   ├── parser_test.go
│   │   ├── lexer.go         # Line-by-line tokenizer
│   │   ├── lexer_test.go
│   │   ├── types.go         # AST node types
│   │   └── title_page.go    # Title page parser
│   ├── renderer/
│   │   ├── pdf.go           # PDF renderer (Courier 12pt)
│   │   ├── pdf_test.go
│   │   └── fonts.go         # Font embedding (Courier)
│   └── elements/
│       └── elements.go      # Element type constants
├── testdata/
│   ├── brick-and-steel.fountain
│   ├── minimal.fountain
│   └── all-elements.fountain
└── README.md
```

## Fase 1: Tipos del AST (`internal/parser/types.go`) - [x] Completada

Definir tipos para cada elemento Fountain:

| Tipo | Descripción |
|------|-------------|
| `TitlePage` | Metadata (Title, Credit, Author, Draft date, Contact) |
| `SceneHeading` | `INT./EXT. LOCATION - TIME` |
| `Action` | Descripción de acción |
| `Character` | Nombre del personaje + extensión opcional |
| `Dialogue` | Texto del personaje |
| `Parenthetical` | `(whispered)`, `(beat)` |
| `Transition` | `CUT TO:`, `FADE OUT.` |
| `Centered` | Texto centrado `>TEXTO<` |
| `Note` | `[NOTA: invisible en output]` |
| `Boneyard` | `/* comentario */` |
| `Section` | `# Heading`, `## Subheading` |
| `Synopsis` | `= Descripción` |
| `Lyric` | `~ texto musical` |
| `Forced` | `@`, `!`, `.` prefix |
| `EmptyLine` | Separador |

**Struct base:**

```go
type Fountain struct {
    TitlePage TitlePage
    Elements  []Element
}
type Element interface{}
```

## Fase 2: Lexer (`internal/parser/lexer.go`) - [x] Completada

Tokenizer line-by-line que identifica:

1. **Title page** — bloque inicial antes de la primera línea vacía que coincida con `Key: value`
2. **Empty lines** — separadores de contexto
3. **Reglas de contexto** — la decisión de tipo depende de:
   - Línea anterior (empty line = contexto limpio)
   - Patrón de la línea (INT./EXT., mayúsculas, `>`, `.`, `@`, `!`, `~`, `=`, `#`, `/*`, `[`, `>...<`)

**Orden de evaluación por línea:**

1. Boneyard (`/* ... */`) — puede multi-línea
2. Note (`[...]`) — puede multi-línea
3. Section (`#`) / Synopsis (`=`)
4. Lyric (`~`)
5. Transition (`TO:`, `>`)
6. Centered (`>...<`)
7. Scene Heading (`.`, INT/EXT patterns)
8. Forced Character (`@`)
9. Forced Action (`!`)
10. Character (uppercase + empty line before)
11. Parenthetical (`(...)`)
12. Dialogue
13. Action (fallback)

## Fase 3: Parser (`internal/parser/parser.go`) - [x] Completada

Convierte tokens en AST:

- Maneja multi-línea: dialogue puede ser múltiples líneas consecutivas
- Boneyard y notes pueden span múltiples líneas
- Sections y synopses son ignoradas en output (v1.1 spec)
- Énfasis: parse `*italics*`, `_underline_`, `**bold**`, `~~strike~~`

## Fase 4: Renderer PDF (`internal/renderer/pdf.go`) - [x] Completada

Usar librería [gofpdf](https://github.com/jung-kurt/gofpdf) para generar PDF.

**Especificación de pantalla:**

| Elemento | Left Margin | Right Edge | Notas |
|----------|-------------|------------|-------|
| Page | 1.5" | 7.5" (1" right) | US Letter |
| Scene Heading | 1.5" | 7.5" | ALL CAPS |
| Action | 1.5" | 7.5" | |
| Character | 3.7" | — | ALL CAPS |
| Parenthetical | 3.1" | ~5" | |
| Dialogue | 2.5" | 6" | |
| Transition | Flush right | 1" | |
| Page number | — | — | Top right, desde page 2 |

**Implementación:**

- Font: Courier 12pt (embebido o nativo)
- Paper: 8.5" × 11" (US Letter)
- Auto page break
- Manejo de page numbers
- Title page separada

## Fase 5: CLI (`main.go`) - [ ] Pendiente

```
USAGE:
  fountain [options] <input.fountain> [output.pdf]

OPTIONS:
  -o, --output    Archivo de salida (default: same name .pdf)
  -h, --help      Mostrar ayuda
  -v, --version   Mostrar versión
```

- Si no se especifica output, genera `input.pdf`
- Lee de stdin si input es `-`
- Exit codes: 0=ok, 1=error

## Fase 6: Tests - [ ] Pendiente

- **Unit tests** para cada tipo de elemento en el parser
- **Integration tests** con archivos `.fountain` reales (Big Fish sample)
- **Golden file tests** — comparar PDF generado con PDF esperado

## Dependencias Go

| Dependencia | Uso |
|-------------|-----|
| `github.com/jung-kurt/gofpdf` | Generación de PDF |

## Orden de Implementación

1. `types.go` — Definir tipos
2. `lexer.go` — Tokenizer básico (scene headings, action, character, dialogue)
3. `parser.go` — Parser con lexer
4. Tests del parser
5. `pdf.go` — Renderer básico
6. Expandir parser a elementos restantes (centered, notes, boneyard, etc.)
7. Title page renderer
8. CLI
9. Tests de integración
10. Docs y polish
