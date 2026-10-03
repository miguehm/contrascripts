# PLAN_GO_REPO.md — exponer posiciones de fuente en el wire format

Objetivo: que el JSON de `parse()` incluya la línea de cada elemento y de
los campos de portada, para que el doble-clic en la vista previa pueda
saltar al offset exacto en el editor Fountain (REVIEW.md punto 4).

Repo: `../2026-08-31-fountain-parser`

## 1. Elementos: `Line` por nodo

- `internal/parser/types.go`: añadir `Line int` a cada struct de elemento:
  `SceneHeading`, `Action`, `Character`, `Dialogue`, `Parenthetical`,
  `Transition`, `Centered`, `Note`, `Boneyard`, `Section`, `Synopsis`,
  `Lyric`, `PageBreak`, `Forced`.
- `internal/parser/parser.go`: poblar `Line` en `nextElement()` desde
  `tok.Line` (lexer.go ya lo emite). Para runs (`Action`, `Dialogue`,
  `Note`, `Boneyard` vía `mergeRun`) usar la línea del primer token del run.
- `internal/parser/json.go`: añadir `Line int` a `Node` **sin omitempty**
  (toda línea emitida es ≥1; omitir el cero es imposible y `omitempty`
  escondería bugs) y mapearlo en `toNode` (todos los casos, incluido
  `PageBreak` y `EmptyLine` filtrado).

## 2. Portada: líneas de los campos

- `internal/parser/json.go`: añadir a `Document`
  `TitlePageLines map[string]int \`json:"titlePageLines,omitempty"\``.
- `internal/parser/parser.go` (`parseTitlePage`/`assignTitleField`):
  registrar la línea del token `Key: value` por clave normalizada
  (misma normalización que ya se aplica al nombre del campo). Poblar el
  mapa en `ToDocument`.

## 3. Tests de fidelidad del wire format

- `internal/parser/json_types_test.go`: declarar y verificar `line` en
  nodos y `titlePageLines` en `Document`.
- `web/tests/package.spec.mjs`: cubrir los nuevos campos del typedef.
- Ajustar fixtures/golden si existen comparaciones de JSON exacto.

## 4. Verificación y publicación

- `go test ./...` en el repo hermano.
- `make dist` (o `make package DIR=<abs>/public/fountain` vía `npm run sync`
  en este repo).
- En este repo: `npm run sync` y comprobar que `src/vendor/fountain.d.mts`
  refleja los nuevos campos (regenerar manualmente el espejo si hace falta).

Cuando esté listo, avisar para verificar en la app que `doc.elements[].line`
y `doc.titlePageLines` llegan al cliente y continuar con la parte de la web.

## Observaciones detectadas al revisar (revisar después)

- `nextElement` en `internal/parser/parser.go` hoy **descarta `Section` y
  `Synopsis`** (devuelve nil), aunque `toNode` en json.go los proyecta. El
  camino Section/Synopsis parece código muerto: o falta cablearlos al AST, o
  sobran en `toNode`. Decisión pendiente — no bloquea este cambio.
- `mergeRun` devuelve solo `[]string`; para sacar la línea del primer token
  basta capturar `p.tokens[p.pos].Line` antes de llamarlo, sin cambiar su
  firma.
- `json_types_test.go` solo valida `TitlePage` contra `fountain.d.ts`;
  `titlePageLines` vive en `Document`, así que no lo cubre. Añadir cobertura
  equivalente para `Node` (campo `line`) y `Document`.
- Recomendación adoptada: `line` siempre presente en el JSON (sin omitempty).
  Reflejado en §1.
