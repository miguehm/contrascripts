// fountain.d.ts — types for the JS wrapper in fountain.mjs.
//
// JavaScript consumers get these from the JSDoc in the .mjs itself; a
// TypeScript one does not, because the wrapper ships as a file a bundler
// compiles rather than as a package with a `types` entry. `make dist` copies
// this next to fountain.mjs so `tsc` finds it by path:
//
//   // tsconfig.json
//   { "compilerOptions": { "paths": { "fountain": ["./fountain/dist/fountain.mjs"] } } }
//
// It is a hand-written mirror of the typedefs above, and nothing compiles the
// two against each other, so two tests keep them honest:
// internal/parser/json_types_test.go checks that every TitlePage field the Go
// wire format emits is declared here, and web/tests/package.spec.mjs checks
// that every typedef in fountain.mjs has a declaration here.

/** A run of inline formatting inside an element. */
export interface Span {
  /** `'text'`, `'bold'` or `'italic'`. */
  type: 'text' | 'bold' | 'italic'
  /** Literal run. Set when `type` is `'text'`. */
  text?: string
  /** Nested spans. Set when `type` is `'bold'` or `'italic'`. */
  content?: Span[]
}

/** Every element name `parse` can return. */
export type ElementType =
  | 'sceneHeading'
  | 'action'
  | 'character'
  | 'dialogue'
  | 'parenthetical'
  | 'transition'
  | 'centered'
  | 'lyric'
  | 'section'
  | 'synopsis'
  | 'pageBreak'
  | 'note'
  | 'boneyard'

/** One screenplay element. */
export interface Node {
  /** Lowercase element name, matching what the PDF draws. */
  type: ElementType
  /** 1-based source line where the element starts. */
  line: number
  /**
   * Element text exactly as parsed, before inline parsing. A dual-dialogue caret
   * is the one exception: it is markup, so it is stripped and recorded as `dual`.
   */
  text?: string
  /** Heading depth. Sections only. */
  level?: number
  /**
   * The cue carried the dual-dialogue caret, which the parser strips from
   * `text` because it is markup rather than part of the name. Nothing draws the
   * two columns yet, so a dual cue prints its name on its own.
   */
  dual?: boolean
  /**
   * Never emitted. A forced element (`.`, `@`, `!`) resolves to the type it
   * stands for, so it arrives as `sceneHeading` with `uppercase`, not as a
   * `prefix`.
   *
   * @deprecated Read `type` instead.
   */
  prefix?: '.' | '!' | '@'
  /** Hint, not a transformation: the renderer must upper-case `text`. */
  uppercase?: boolean
  /** Hint, not a transformation: the renderer must style the whole element italic. */
  baseItalic?: boolean
  /** Parsed `**bold**` / `*italic*` runs, present only where the PDF honours them. */
  inline?: Span[]
}

/**
 * Title page metadata.
 *
 * Every field is always present — the Go wire format has no `omitempty` here,
 * so an absent key is an empty string. Keys are normalized, which is why they
 * are camelCase: `draft-date`, `draft_date` and `Draft date` all land in
 * `draftDate`.
 */
export interface TitlePage {
  title: string
  credit: string
  author: string
  draftDate: string
  contact: string
  source: string
  revision: string
  logline: string
  tagline: string
  synopsis: string
  genre: string
  subgenre: string
  tone: string
  themes: string
  comps: string
  format: string
  season: string
  episode: string
  episodeTitle: string
  runtime: string
  pages: string
  language: string
  status: string
  prodCompany: string
  producer: string
  director: string
  budget: string
  adaptedBy: string
  basedOn: string
  copyright: string
  wga: string
  rights: string
  dedication: string
  disclaimer: string
  draftType: string
  version: string
  revisionColor: string
  revisionDate: string
  email: string
  phone: string
  agent: string
  manager: string
  address: string
  /** Whether the script asked for scene numbers. Off unless set. */
  sceneNumbers: boolean
  /**
   * Production script: emit the pagination notation a spec script omits,
   * namely (MORE), (CONT'D) and the scene CONTINUED marks.
   */
  production: boolean
  /** Keys the parser did not recognise, rendered on the title page too. */
  custom: Record<string, string>
}

/** A parsed screenplay. */
export interface Document {
  titlePage: TitlePage
  /** Source line of each recognized title page key, keyed by normalized name ("draft date"). */
  titlePageLines?: Record<string, number>
  /** The screenplay, empty lines dropped. */
  elements: Node[]
}

/** A format warning from `lint`. */
export interface Warning {
  /** 1-based line number in the source. */
  line: number
  /** Stable machine-readable code, e.g. `'SCENE_NO_TIME'`. */
  code: string
  /** Human-readable description. */
  message: string
}

/** Options accepted by {@link createFountain}. */
export interface Options {
  /**
   * Directory holding the `.wasm` files, resolved against the page. A
   * site-absolute path such as `'/fountain'` is the normal case under a
   * bundler, where this module is compiled into a chunk and its own
   * `import.meta.url` is no longer a sibling of the `.wasm`. Defaults to the
   * directory containing this module.
   */
  base?: string | URL
  /** Path to `wasm_exec.js`, resolved against `base`. Defaults to `'wasm_exec.js'`. */
  wasmExec?: string | URL
  /**
   * Load `wasm_exec.js` by appending a `<script>` tag. Set to `false` if the
   * page already includes it — under a strict CSP, for instance. Default
   * `true`.
   */
  injectWasmExec?: boolean
}

/** The scripting API returned by {@link createFountain}. */
export interface Fountain {
  /** Parse to a Document. Synchronous, and it blocks the main thread. */
  parse(text: string): Document
  /** Format warnings. Synchronous, and it blocks the main thread. */
  lint(text: string): Warning[]
  /** Render to a PDF, loading the heavy module on first use. */
  renderPDF(text: string): Promise<Uint8Array>
  /** Whether the PDF module is already resident. */
  pdfLoaded(): boolean
  /** Warm the PDF module up early. */
  preloadPDF(): Promise<void>
  /** Set if a module died after loading; every later call rethrows it. */
  readonly failure: Error | null
}

/**
 * A parse, lint or render failure. The Go side reports these by logging to the
 * console and returning `null`; this turns that into something catchable.
 *
 * It is also what `createFountain` throws when there is no `document` to boot
 * the runtime in.
 */
export declare class FountainError extends Error {
  constructor(message: string, cause?: unknown)
}

/**
 * Boot the parser module and return the scripting API. Rejects with a
 * {@link FountainError} outside the browser.
 *
 * @param options see {@link Options}
 */
export declare function createFountain(options?: Options): Promise<Fountain>

export default createFountain
