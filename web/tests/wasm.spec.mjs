import { test, expect } from '@playwright/test'
import { readFile } from 'node:fs/promises'

const SCRIPT = await readFile(
  new URL('../../testdata/screenplay-complete.fountain', import.meta.url),
  'utf8',
)

// A scene heading with no " - TIME" is the canonical lint warning, so this does
// not depend on the fixture's incidental formatting.
const LINTABLE = 'INT. HOUSE\n\nA man enters the room.\n'

test.describe.configure({ mode: 'serial' })

/** @type {import('@playwright/test').Page} */
let page
const pageErrors = []
const consoleErrors = []

test.beforeAll(async ({ browser }) => {
  page = await browser.newPage()
  page.on('pageerror', (err) => pageErrors.push(String(err)))
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text())
  })

  await page.goto('/web/harness.html')

  // No arguments on purpose: the wrapper must resolve its own .wasm siblings
  // out of /dist/, which is the layout `make dist` actually produces.
  await page.evaluate(async () => {
    const { createFountain } = await import('/dist/fountain.mjs')
    window.fountain = await createFountain()
  })
})

test.afterAll(async () => {
  await page?.close()
})

test('parses a screenplay into a Document', async () => {
  const doc = await page.evaluate((text) => window.fountain.parse(text), SCRIPT)

  expect(doc.titlePage.title).toBe('THE LAST SIGNAL')
  expect(doc.titlePage.author).toBe('Jane Smith')
  expect(Array.isArray(doc.elements)).toBe(true)
  expect(doc.elements.length).toBeGreaterThan(0)

  const scenes = doc.elements.filter((el) => el.type === 'sceneHeading')
  expect(scenes.length).toBeGreaterThan(0)
  expect(scenes[0].text).toBe('EXT. SPACE - NIGHT')
  expect(scenes[0].uppercase).toBe(true)
})

test('keeps inline formatting as nested spans', async () => {
  const doc = await page.evaluate(() =>
    window.fountain.parse('EXT. HOUSE - DAY\n\nA **loud** bang.\n'),
  )

  const action = doc.elements.find((el) => el.type === 'action')
  expect(action.text).toBe('A **loud** bang.')
  expect(action.inline.map((s) => s.type)).toEqual(['text', 'bold', 'text'])
  expect(action.inline[0].text).toBe('A ')
  expect(action.inline[1].content[0]).toEqual({ type: 'text', text: 'loud' })
  expect(action.inline[2].text).toBe(' bang.')
})

test('lints without touching the parser path', async () => {
  const warnings = await page.evaluate((text) => window.fountain.lint(text), LINTABLE)

  expect(Array.isArray(warnings)).toBe(true)
  expect(warnings.map((w) => w.code)).toContain('SCENE_NO_TIME')
  expect(warnings[0]).toMatchObject({ line: expect.any(Number), message: expect.any(String) })
})

test('lint returns an empty array, not null, for a clean script', async () => {
  // A consumer writes warnings.forEach(...) without a guard, so a clean script
  // must produce [] at the seam, not the JSON null a nil Go slice would encode.
  const warnings = await page.evaluate(() =>
    window.fountain.lint('Title: T\nAuthor: A\n\nEXT. CASA - DIA\n\nTexto.\n'),
  )

  expect(Array.isArray(warnings)).toBe(true)
  expect(warnings).toEqual([])
})

test('has not loaded the PDF module yet', async () => {
  // The split into two binaries only pays off if the heavy one stays off the
  // first paint, so laziness is asserted rather than assumed.
  expect(await page.evaluate(() => window.fountain.pdfLoaded())).toBe(false)
})

test('renders a PDF on demand', async () => {
  const info = await page.evaluate(async (text) => {
    const bytes = await window.fountain.renderPDF(text)
    return {
      type: bytes.constructor.name,
      length: bytes.length,
      magic: String.fromCharCode(...bytes.slice(0, 5)),
    }
  }, SCRIPT)

  expect(info.type).toBe('Uint8Array')
  expect(info.length).toBeGreaterThan(1000)
  expect(info.magic).toBe('%PDF-')
  expect(await page.evaluate(() => window.fountain.pdfLoaded())).toBe(true)
})

test('the parser survives the PDF module entering the same page', async () => {
  // The reason the spec is one serial run on one page. Two compiled modules
  // means two Go runtimes, and each patches the same globalThis. If the second
  // one clobbered the first, this is where it would show.
  const result = await page.evaluate((text) => {
    const doc = window.fountain.parse(text)
    return {
      title: doc.titlePage.title,
      elements: doc.elements.length,
      parseStillAGlobal: typeof globalThis.fountainParse,
      renderStillAGlobal: typeof globalThis.fountainRenderPDF,
    }
  }, SCRIPT)

  expect(result.title).toBe('THE LAST SIGNAL')
  expect(result.elements).toBeGreaterThan(0)
  expect(result.parseStillAGlobal).toBe('function')
  expect(result.renderStillAGlobal).toBe('function')
})

test('the Go globals signal failure with null', async () => {
  // The wrapper's null check is defensive: through its public API it always
  // passes exactly one argument, so the error path is unreachable by
  // construction. These assertions pin the contract that check defends
  // against — if the Go side starts throwing, or returns an empty string
  // instead, the wrapper's error handling has to change with it.
  //
  // Each call logs to console.error on the way out, which the final test
  // accounts for.
  const outcomes = await page.evaluate(() => ({
    parse: globalThis.fountainParse(),
    lint: globalThis.fountainLint(),
    render: globalThis.fountainRenderPDF(),
  }))

  expect(outcomes).toEqual({ parse: null, lint: null, render: null })
})

test('boots without page errors or unexpected console errors', () => {
  const expected = /missing text argument/
  expect(pageErrors).toEqual([])
  expect(consoleErrors.filter((text) => !expected.test(text))).toEqual([])
})
