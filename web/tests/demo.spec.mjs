import { test, expect } from '@playwright/test'

// The API spec proves the modules work. This one proves the documented usage
// works, by loading the page a consumer would copy from.

test('the demo page boots, previews and exports', async ({ page }) => {
  const errors = []
  page.on('pageerror', (err) => errors.push(String(err)))

  await page.goto('/web/index.html')

  await expect(page.locator('#status')).toHaveText('parser ready')

  // The preview is rendered from the Document, not the raw text.
  await expect(page.locator('#script .sceneHeading').first()).toHaveText(
    'EXT. SPACE - NIGHT',
  )
  await expect(page.locator('#script .character').first()).toHaveText('ELENA')
  await expect(page.locator('#script b')).toHaveText('lone')

  // The seeded script is lint-clean, so the panel starts empty.
  await expect(page.locator('#warnings li')).toHaveCount(0)

  // Editing re-renders, and a script with a missing time-of-day warns.
  await page.locator('#source').fill('INT. HOUSE\n\nSomeone waits.\n')
  await expect(page.locator('#script .sceneHeading').first()).toHaveText('INT. HOUSE')
  await expect(page.locator('#warnings li').first()).toContainText('SCENE_NO_TIME')

  // The PDF module is still unloaded until the button is pressed.
  await expect(page.locator('#status')).toHaveText('parser ready')

  const download = page.waitForEvent('download')
  await page.locator('#pdf').click()
  expect((await download).suggestedFilename()).toBe('script.pdf')

  expect(errors).toEqual([])
})
