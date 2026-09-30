import { test, expect } from '@playwright/test'

// The API spec proves the modules work. This one proves the documented usage
// works, by loading the page a consumer would copy from.

// These two pin the paths the server and index.html tell a person to open. The
// spec below reaches index.html directly, so nothing else would notice if
// serving a directory broke again — which it did: both routes answered a bare
// 404 for a page that loads fine.

test('the repository root redirects to the preview', async ({ page }) => {
  const errors = []
  page.on('pageerror', (err) => errors.push(String(err)))

  await page.goto('/')

  await expect(page.locator('#status')).toHaveText('ready')
  expect(errors).toEqual([])
})

test('a directory serves its index.html', async ({ page }) => {
  const errors = []
  page.on('pageerror', (err) => errors.push(String(err)))

  // The path index.html documents, and the one / used to land on after the
  // redirect above.
  await page.goto('/web/')

  await expect(page.locator('#status')).toHaveText('ready')
  expect(errors).toEqual([])
})

/**
 * The bytes the iframe is showing, read back through its blob URL.
 *
 * Asserting on the bytes is what makes this stronger than the DOM assertions
 * it replaces. Those checked that a <p class="character"> existed holding
 * "ELENA", which said nothing about whether the rendered output was right. This
 * checks that what is on screen came out of the renderer at all — and it does
 * not depend on Chromium's PDF plugin working in headless.
 */
const shown = (page) =>
  page.evaluate(async () => {
    const src = document.getElementById('preview').src
    if (!src.startsWith('blob:')) return { src, magic: null, length: 0 }
    const buf = new Uint8Array(await (await fetch(src)).arrayBuffer())
    return { src, magic: String.fromCharCode(...buf.slice(0, 5)), length: buf.length }
  })

test('the demo page previews the renderer output and exports it', async ({ page }) => {
  const errors = []
  page.on('pageerror', (err) => errors.push(String(err)))

  await page.goto('/web/index.html')

  // 'ready' is only set once the first renderPDF has resolved, so it doubles as
  // the assertion that the preview is populated.
  await expect(page.locator('#status')).toHaveText('ready')

  const first = await shown(page)
  expect(first.magic).toBe('%PDF-')
  // The seeded script is a title page plus a body page of Courier Prime, so a
  // real render is tens of kilobytes, not the couple a stub would produce.
  expect(first.length).toBeGreaterThan(10_000)

  // The seeded script is lint-clean, so the panel starts empty.
  await expect(page.locator('#warnings li')).toHaveCount(0)

  // Editing re-renders, and a script with a missing time-of-day warns.
  await page.locator('#source').fill('INT. HOUSE\n\nSomeone waits.\n')
  await expect(page.locator('#warnings li').first()).toContainText('SCENE_NO_TIME')

  // The blob is a new object, so the URL changes. That is the proof a render
  // happened: a stale preview would leave the URL untouched. Waited on rather
  // than slept for, so it cannot pass on a machine slower than this one.
  await page.waitForFunction(
    (before) => document.getElementById('preview').src !== before,
    first.src,
  )
  await expect(page.locator('#status')).toHaveText('ready')

  const second = await shown(page)
  expect(second.src).not.toBe(first.src)
  expect(second.magic).toBe('%PDF-')
  // A one-scene script is shorter than the seeded title page plus two scenes.
  expect(second.length).toBeLessThan(first.length)

  const download = page.waitForEvent('download')
  await page.locator('#pdf').click()
  expect((await download).suggestedFilename()).toBe('script.pdf')

  expect(errors).toEqual([])
})
