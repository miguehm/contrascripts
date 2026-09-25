import { test, expect } from '@playwright/test'

// Integration test for the way a bundler-based frontend actually consumes the
// package. web/tests/wasm.spec.mjs covers the modules themselves; this covers
// the seam, where the failure modes are entirely different: a 404 on the shim,
// the wrapper resolving its .wasm against the wrong URL, and a React app
// booting one Go runtime per mount.

const BOOTED = 'ready'

/** Collects every network failure and non-2xx/3xx the page produces. */
function watch(page) {
  const failed = []
  const bad = []
  const consoleErrors = []

  page.on('requestfailed', (req) =>
    failed.push(`${req.url()} — ${req.failure()?.errorText ?? 'failed'}`),
  )
  page.on('response', (res) => {
    if (res.status() >= 400) bad.push(`${res.status()} ${res.url()}`)
  })
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text())
  })

  return { failed, bad, consoleErrors, urls: [] }
}

test('boots once, parses, lints and exports', async ({ page }) => {
  const net = watch(page)
  const requested = []
  page.on('request', (req) => requested.push(req.url()))

  await page.goto('/')
  await expect(page.locator('#status')).toHaveText(BOOTED)

  // The seeded script parses into elements and trips the missing-time warning.
  await expect(page.locator('#count')).not.toHaveText('0 elements')
  await expect(page.locator('#warnings li').first()).toContainText('SCENE_NO_TIME')

  // StrictMode double-invokes effects, so a naive createFountain() in an effect
  // would leave two Go runtimes on globalThis.
  expect(await page.evaluate(() => window.__fountainBoots)).toBe(1)

  // The assets were fetched from the staged public/fountain folder at their
  // own URLs, which is what the explicit `base` option points at. The wrapper
  // is deliberately not on this list: it is compiled into the bundle, and
  // `make package` does not stage a second copy of it into public/.
  for (const file of ['/fountain/wasm_exec.js', '/fountain/fountain-parser.wasm']) {
    expect(
      requested.some((url) => url.includes(file)),
      `expected ${file} to be requested, got ${JSON.stringify(requested)}`,
    ).toBe(true)
  }
  expect(
    requested.some((url) => url.includes('/fountain/fountain.mjs')),
    'the wrapper was served from public/ instead of compiled into the bundle',
  ).toBe(false)

  // The heavy module is fetched when the button is pressed, not on first paint.
  expect(requested.some((url) => url.includes('/fountain/fountain-pdf.wasm'))).toBe(false)

  const download = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Download PDF' }).click()
  expect((await download).suggestedFilename()).toBe('script.pdf')
  expect(requested.some((url) => url.includes('/fountain/fountain-pdf.wasm'))).toBe(true)

  expect(net.failed, 'requests failed').toEqual([])
  expect(net.bad, 'non-2xx responses').toEqual([])
  expect(net.consoleErrors, 'console errors').toEqual([])
})

test('a failed load surfaces in the UI rather than hanging on "booting"', async ({
  page,
}) => {
  // Regression guard for the promise bookkeeping: if loadFountain() rejected and
  // the component swallowed it, the status would sit at "booting…" forever.
  await page.route('**/fountain/fountain-parser.wasm', (route) => route.abort())
  await page.goto('/')
  await expect(page.locator('#status')).not.toHaveText('booting…')
  await expect(page.locator('#status.error')).toBeVisible()
})
