import { test, expect } from '@playwright/test'
import { readFile } from 'node:fs/promises'

// The packaging contract, and the two guards that keep it from rotting: what
// `make dist` has to ship, and what the wrapper is allowed to do outside a
// browser.
//
// The rest of the smoke suite proves the modules run. This one runs in Node —
// no page — because the failures it looks for are exactly the ones a browser
// cannot show: a missing file in dist/, a stale copy, a package manifest that
// points at nothing, and a wrapper that would be imported during a prerender
// and explode there.

const DIST = new URL('../../dist/', import.meta.url)
const WRAPPER = new URL('../fountain.mjs', import.meta.url)
const TYPES = new URL('../fountain.d.ts', import.meta.url)

/** Files `make dist` must produce, and a consumer cannot work without. */
const REQUIRED = [
  'fountain.mjs',
  'fountain.d.ts',
  'wasm_exec.js',
  'fountain-parser.wasm',
  'fountain-pdf.wasm',
  'manifest.json',
  'package.json',
]

const readDist = (name) => readFile(new URL(name, DIST), 'utf8')

test('dist ships the whole package', async () => {
  // fountain.d.ts and package.json joined the list for TypeScript consumers:
  // without them the wrapper is untyped and the folder is not a package, so a
  // bundler resolving it by path gets `any` and no `exports` map.
  for (const file of REQUIRED) {
    await expect(readDist(file), `dist/${file} is missing`).resolves.not.toBe('')
  }
})

test('dist is a current copy of the wrapper and its types', async () => {
  // dist/ is generated, so a stale copy is a silent one: the browser suite
  // would keep testing yesterday's wrapper and nobody would notice until a
  // consumer hit the bug the fix was for.
  for (const [name, source] of [
    ['fountain.mjs', WRAPPER],
    ['fountain.d.ts', TYPES],
  ]) {
    expect(await readDist(name), `dist/${name} is stale; run make dist`).toBe(
      await readFile(source, 'utf8'),
    )
  }
})

test('the package manifest points at files that exist', async () => {
  const pkg = JSON.parse(await readDist('package.json'))

  expect(pkg.type).toBe('module')
  expect(pkg.types).toBe('./fountain.d.ts')

  // Every specifier in `exports` is a path into dist/. A typo here fails at
  // resolution time in the consumer's build, which is the worst place to find
  // it out.
  const targets = new Set()
  const collect = (node) => {
    if (typeof node === 'string') return targets.add(node)
    if (node && typeof node === 'object') Object.values(node).forEach(collect)
  }
  collect(pkg.exports)
  collect(pkg.main)
  collect(pkg.types)

  expect(targets.size).toBeGreaterThan(3)
  for (const target of targets) {
    if (target.includes('*')) continue
    await expect(readDist(target), `package.json points at ${target}`).resolves.not.toBe('')
  }
})

test('the wire typedef declares source line fields', async () => {
  // Node.line and Document.titlePageLines carry the source positions the
  // editor double-click feature needs. If the hand-written .d.ts drops them,
  // a TS consumer compiles against fields the runtime actually sends.
  const types = await readFile(TYPES, 'utf8')

  const nodeBody = types.slice(types.indexOf('export interface Node'))
  expect(nodeBody).toContain('line: number')

  const docBody = types.slice(types.indexOf('export interface Document'))
  expect(docBody).toContain('titlePageLines')
})

test('the paginate typedef declares scene page fields', async () => {
  // ScenePage carries the line → absolute PDF page map the preview's exact
  // scene jump needs. The wrapper documents it in JSDoc and the .d.ts must
  // mirror both the type and the Fountain.paginate method.
  const types = await readFile(TYPES, 'utf8')
  const wrapper = await readFile(WRAPPER, 'utf8')

  const sceneBody = types.slice(types.indexOf('export interface ScenePage'))
  expect(sceneBody).toContain('line: number')
  expect(sceneBody).toContain('page: number')

  const fountainBody = types.slice(types.indexOf('export interface Fountain'))
  expect(fountainBody).toContain('paginate')

  expect(wrapper).toContain('fountainPaginate')
})

test('the manifest version is the CLI version', async () => {
  // One source of truth: cmd/fountain/main.go. `make dist` reads the constant
  // out of it, and a consumer reading package.json must not learn a different
  // version than `fountain -v` reports.
  const main = await readFile(new URL('../../cmd/fountain/main.go', import.meta.url), 'utf8')
  const version = main.match(/^const version = "([^"]+)"/m)?.[1]
  expect(version, 'the version constant moved; update the test').toBeTruthy()

  const pkg = JSON.parse(await readDist('package.json'))
  expect(pkg.version).toBe(version)
})

test('every typedef in the wrapper is declared in the .d.ts', async () => {
  // The .d.ts is a hand-written mirror: the .mjs documents its types in JSDoc,
  // which TypeScript cannot read from a .mjs, and the .d.ts is what a TS
  // consumer actually gets. Nothing compiles the two against each other, so a
  // type added to one and forgotten in the other is invisible from here.
  const wrapper = await readFile(WRAPPER, 'utf8')
  const types = await readFile(TYPES, 'utf8')

  const declared = new Set()
  for (const match of types.matchAll(
    /^export (?:declare )?(?:interface|class|function|type) (\w+)/gm,
  )) {
    declared.add(match[1])
  }

  const documented = new Set()
  for (const match of wrapper.matchAll(/@typedef\s+\{[^}]*\}\s*(\w+)/g)) {
    documented.add(match[1])
  }
  for (const match of wrapper.matchAll(/^export (?:async )?(?:class|function) (\w+)/gm)) {
    documented.add(match[1])
  }

  expect(documented.size).toBeGreaterThan(4)
  for (const name of documented) {
    expect(declared, `${name} is documented in fountain.mjs but not in fountain.d.ts`).toContain(
      name,
    )
  }
})

test.describe('outside the browser', () => {
  test('createFountain rejects with an actionable error', async () => {
    // An SSR pass or a prerender imports this module and calls it where there
    // is no document. Left alone it fails as a fetch against a file:// URL, or
    // worse as a 404 on the shim, and neither mentions the cause — which is
    // how a webpack-time guess turns into an afternoon.
    const { createFountain, FountainError } = await import('../fountain.mjs')

    expect(globalThis.Go).toBeUndefined()
    expect(globalThis.document).toBeUndefined()

    await expect(createFountain()).rejects.toThrow(FountainError)
    await expect(createFountain()).rejects.toThrow(/browser module/)
    await expect(createFountain()).rejects.toThrow(/ssr: false/)
  })

  test('the guard fires before any fetch, whatever base is passed', async () => {
    // The default base is this module's own URL, which under Node is a file://
    // path. An explicit site-absolute base is the case that looks most likely
    // to work, and is the one a Next.js app passes.
    const { createFountain } = await import('../fountain.mjs')

    for (const base of [undefined, '/fountain', 'https://cdn.example.com/fountain/']) {
      await expect(createFountain(base === undefined ? {} : { base })).rejects.toThrow(
        /cannot boot without a document/,
      )
    }
  })
})
