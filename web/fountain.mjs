/**
 * fountain.mjs — JS wrapper around the two js/wasm modules built by this repo.
 *
 * The raw modules are not a library you can `import`: they are standalone
 * WebAssembly binaries that need Go's runtime shim (wasm_exec.js, version-locked
 * to the toolchain that compiled them) and they register their API as globals on
 * `globalThis`. Everything this file does is glue a web project would otherwise
 * have to rewrite:
 *
 *   - boots wasm_exec.js and reports a useful error if it is missing or stale
 *   - calls the Go globals, which return JSON *strings*, and parses them
 *   - turns the `null` + console.error error convention into a thrown error
 *   - loads fountain-pdf.wasm on demand instead of on first paint
 *
 * Usage:
 *
 *   import { createFountain } from './fountain.mjs'
 *   const fountain = await createFountain()          // resolves next to this file
 *   const doc = fountain.parse(text)                 // sync
 *   const warnings = fountain.lint(text)             // sync
 *   const pdf = await fountain.renderPDF(text)       // async, lazy 6.7 MB load
 *
 * @module fountain
 */

/**
 * A parse, lint or render failure. The Go side reports these by logging to the
 * console and returning null; this class turns that into something catchable.
 */
export class FountainError extends Error {
  /** @param {string} message @param {unknown} [cause] */
  constructor(message, cause) {
    super(message)
    this.name = 'FountainError'
    if (cause !== undefined) this.cause = cause
  }
}

/**
 * @typedef {object} Span
 * @property {'text'|'bold'|'italic'} type
 * @property {string} [text] literal run, set when type is 'text'
 * @property {Span[]} [content] nested spans, set for 'bold' and 'italic'
 */

/**
 * @typedef {object} Node
 * @property {string} type element name, e.g. 'action' or 'sceneHeading'
 * @property {string} [text] element text before inline parsing
 * @property {number} [level] heading depth, sections only
 * @property {string} [prefix] forcing character ('.', '!', '@')
 * @property {boolean} [uppercase] renderer must upper-case `text`
 * @property {boolean} [baseItalic] renderer must style the whole element italic
 * @property {Span[]} [inline] parsed **bold** / *italic* runs
 */

/**
 * @typedef {object} Document
 * @property {Record<string, any>} titlePage normalized title page metadata
 * @property {Node[]} elements the screenplay, empty lines dropped
 */

/**
 * @typedef {object} Warning
 * @property {number} line 1-based line number in the source
 * @property {string} code stable machine-readable code, e.g. 'SCENE_NO_TIME'
 * @property {string} message human-readable description
 */

/**
 * @typedef {object} Fountain
 * @property {(text: string) => Document} parse parse to a Document
 * @property {(text: string) => Warning[]} lint format warnings
 * @property {(text: string) => Promise<Uint8Array>} renderPDF render to a PDF
 * @property {() => boolean} pdfLoaded whether the PDF module is already resident
 * @property {() => Promise<void>} preloadPDF warm the PDF module up early
 * @property {Error|null} [failure] set if a module died after loading
 */

/**
 * @typedef {object} Options
 * @property {string|URL} [base] directory holding the .wasm files. A
 *   site-absolute path such as '/fountain' is the normal case under a bundler,
 *   where this module is compiled into a chunk and its own import.meta.url is
 *   no longer a sibling of the .wasm. Defaults to the directory containing
 *   this module.
 * @property {string|URL} [wasmExec] path to wasm_exec.js, resolved against
 *   `base`; defaults to `wasm_exec.js` in that directory. Ignored if
 *   `globalThis.Go` already exists.
 * @property {boolean} [injectWasmExec] when true, load wasm_exec.js by appending
 *   a <script> tag. Set to false if the host page already includes it and you
 *   only need the reference. Default true.
 */

const GLOBALS = {
  parse: 'fountainParse',
  lint: 'fountainLint',
  render: 'fountainRenderPDF',
}

/** The directory this module was served from, so defaults need no arguments. */
const HERE = new URL('.', import.meta.url)

/**
 * Ensures Go's runtime shim is present, loading it over the network if not.
 *
 * wasm_exec.js must be the copy from the same Go release that compiled the
 * .wasm. A mismatched one does not fail loudly at build time — it fails when
 * the runtime initialises, with an error that says nothing about the cause, so
 * the mismatch is reported here instead.
 *
 * @param {string|URL} src
 * @param {string|URL} base directory src is resolved against
 * @returns {Promise<typeof globalThis.Go>}
 */
async function ensureGo(src, base) {
  if (globalThis.Go) return globalThis.Go

  const url = new URL(src, base).href
  await new Promise((resolve, reject) => {
    const el = document.createElement('script')
    el.src = url
    el.onload = resolve
    el.onerror = () =>
      reject(
        new FountainError(
          `could not load ${url}. It ships in dist/ next to the .wasm files; ` +
            `run "make dist" in the fountain repo.`,
        ),
      )
    document.head.appendChild(el)
  })

  if (!globalThis.Go) {
    throw new FountainError(
      `${url} loaded but did not define globalThis.Go. It is probably not the ` +
        `Go wasm_exec.js, or it is newer than the Go that built the .wasm.`,
    )
  }
  return globalThis.Go
}

/**
 * Instantiates a module, preferring streaming and falling back when the server
 * does not send the application/wasm content type that streaming requires.
 *
 * @param {string} name file name within the package directory
 * @param {WebAssembly.Imports} imports
 * @param {URL} base absolute URL of the package directory
 * @returns {Promise<WebAssembly.Instance>}
 */
async function instantiate(name, imports, base) {
  const href = new URL(name, base).href

  if (WebAssembly.instantiateStreaming && /^https?:/.test(href)) {
    try {
      const streamed = await WebAssembly.instantiateStreaming(fetch(href), imports)
      return streamed.instance
    } catch {
      // Fall through: a wrong Content-Type or a 404 on the wasm path.
    }
  }

  const response = await fetch(href)
  if (!response.ok) {
    throw new FountainError(`${href} returned HTTP ${response.status}`)
  }
  const { instance } = await WebAssembly.instantiate(await response.arrayBuffer(), imports)
  return instance
}

/**
 * Starts a Go module and hands back the globals it registered.
 *
 * go.run() is deliberately not awaited: main() ends in `select{}`, so the
 * promise never settles. A Go panic still surfaces as a thrown exception on the
 * call itself; a module that dies for any other reason shows up as a rejection
 * here, which is recorded so the wrapper can report it later instead of leaving
 * an unhandled rejection behind.
 *
 * @param {WebAssembly.Instance} instance
 * @param {Go} go
 * @param {string[]} names globals expected on globalThis
 * @returns {Promise<{calls: Record<string, Function>, exited: Promise<never>}>}
 */
async function start(instance, go, names) {
  const exited = go.run(instance)
  exited.catch(() => {
    /* surfaced through Fountain.failure */
  })

  const calls = {}
  for (const name of names) {
    const fn = globalThis[name]
    if (typeof fn !== 'function') {
      throw new FountainError(
        `the module did not register globalThis.${name}. ` +
          `Check that the .wasm matches the globals this wrapper expects.`,
      )
    }
    calls[name] = fn
  }
  return { calls, exited }
}

/**
 * Reads a Go global that returns a JSON string, or null when it failed.
 *
 * @param {Function} fn
 * @param {string} name
 * @param {string} text
 * @returns {any}
 */
function readJSON(fn, name, text) {
  const raw = fn(text)
  if (raw === null || raw === undefined) {
    throw new FountainError(
      `${name}() failed inside the wasm module; the reason was logged to the ` +
        `browser console.`,
    )
  }
  return JSON.parse(raw)
}

/**
 * Boots the parser module and returns the scripting API.
 *
 * Only the parser is loaded here. `fountain-pdf.wasm` carries gofpdf and four
 * embedded Courier Prime faces — roughly triple the parser — so it stays out of
 * the first paint and is fetched by `renderPDF()` on demand.
 *
 * @param {Options} [options]
 * @returns {Promise<Fountain>}
 */
export async function createFountain(options = {}) {
  const {
    base = HERE,
    wasmExec = 'wasm_exec.js',
    injectWasmExec = true,
  } = options

  // The shim is resolved against `base`, not against this module's own URL.
  // Under a bundler the .mjs is compiled into an output chunk and
  // import.meta.url stops pointing at the folder holding the .wasm files, so
  // resolving against it would 404. When `base` is left at its default the two
  // are the same thing and nothing changes.
  //
  // `base` is normalised to an absolute directory URL first, because a caller
  // passing a site-absolute path like '/fountain' is the normal case under a
  // bundler and new URL(relative, relative) throws. Resolving it against the
  // page covers that, and leaves absolute and URL-object inputs untouched.
  //
  // The trailing slash is load-bearing: URL resolves 'wasm_exec.js' against
  // '/fountain' as a sibling of a *file* and yields '/wasm_exec.js', but
  // against '/fountain/' as a file inside a *directory*. The default HERE
  // already ends in a slash, so this only ever affects an explicit base.
  const baseURL = new URL(base, globalThis.location?.href ?? HERE)
  if (!baseURL.pathname.endsWith('/')) baseURL.pathname += '/'
  const go = injectWasmExec ? await ensureGo(wasmExec, baseURL) : globalThis.Go
  if (!go) throw new FountainError('no Go runtime available and injection is disabled')

  /** @type {Error|null} */
  let failure = null

  // One Go instance for both the imports and run(): a second instance would be
  // a second runtime whose import object describes a heap that is not the one
  // the module ends up running on.
  const runtime = new go()
  const parser = await start(
    await instantiate('fountain-parser.wasm', runtime.importObject, baseURL),
    runtime,
    [GLOBALS.parse, GLOBALS.lint],
  )
  parser.exited.catch((err) => {
    failure ||= err instanceof Error ? err : new FountainError(String(err))
  })

  /** @type {Promise<Function>|null} */
  let pdf = null

  async function loadPDF() {
    if (pdf) return pdf
    pdf = (async () => {
      const pdfRuntime = new go()
      const started = await start(
        await instantiate('fountain-pdf.wasm', pdfRuntime.importObject, baseURL),
        pdfRuntime,
        [GLOBALS.render],
      )
      started.exited.catch((err) => {
        failure ||= err instanceof Error ? err : new FountainError(String(err))
      })
      return started.calls[GLOBALS.render]
    })()
    try {
      return await pdf
    } catch (err) {
      // Do not cache a failed load, so a later renderPDF can retry.
      pdf = null
      throw err
    }
  }

  return {
    parse: (text) => {
      if (failure) throw failure
      return readJSON(parser.calls[GLOBALS.parse], 'parse', text)
    },
    lint: (text) => {
      if (failure) throw failure
      return readJSON(parser.calls[GLOBALS.lint], 'lint', text)
    },
    renderPDF: async (text) => {
      if (failure) throw failure
      const render = await loadPDF()
      const bytes = render(text)
      if (bytes === null || bytes === undefined) {
        throw new FountainError(
          'renderPDF() failed inside the wasm module; the reason was logged ' +
            'to the browser console.',
        )
      }
      return bytes
    },
    pdfLoaded: () => pdf !== null,
    preloadPDF: async () => {
      await loadPDF()
    },
    get failure() {
      return failure
    },
  }
}

export default createFountain
