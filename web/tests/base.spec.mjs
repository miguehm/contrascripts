import { test, expect } from '@playwright/test'

// The bundler case, exercised without a bundler. web/vite covers the real
// thing, but these are the two failure modes that cost the most time to find
// and deserve a fast check that does not need a build step:
//
//   - a site-absolute `base` like '/dist' is not a valid URL base
//   - without a trailing slash it resolves 'wasm_exec.js' as a sibling of a
//     file, i.e. '/wasm_exec.js', and the shim silently fails to load
//
// Both produced the same unhelpful symptom, so they are pinned here.

test('accepts an explicit site-absolute base', async ({ page }) => {
  const errors = []
  page.on('pageerror', (err) => errors.push(String(err)))

  await page.goto('/web/harness.html')

  const result = await page.evaluate(async () => {
    const { createFountain } = await import('/dist/fountain.mjs')
    // No trailing slash on purpose.
    const fountain = await createFountain({ base: '/dist' })
    return {
      title: fountain.parse('Title: T\n\nEXT. CASA - DIA\n\nTexto.\n').titlePage.title,
      warnings: fountain.lint('INT. HOUSE\n\nTexto.\n').length,
    }
  })

  expect(result.title).toBe('T')
  expect(result.warnings).toBeGreaterThan(0)
  expect(errors).toEqual([])
})

test('the PDF module loads from an explicit base too', async ({ page }) => {
  await page.goto('/web/harness.html')

  const magic = await page.evaluate(async () => {
    const { createFountain } = await import('/dist/fountain.mjs')
    const fountain = await createFountain({ base: '/dist/' })
    const bytes = await fountain.renderPDF('Title: T\n\nEXT. CASA - DIA\n\nTexto.\n')
    return String.fromCharCode(...bytes.slice(0, 5))
  })

  expect(magic).toBe('%PDF-')
})

test('uses a wasm_exec.js the page already loaded', async ({ page }) => {
  // The escape hatch the README documents for a CSP that forbids injecting
  // scripts. It is a separate code path from ensureGo, which is the branch
  // every other test here takes, so nothing else would notice it breaking.
  const errors = []
  page.on('pageerror', (err) => errors.push(String(err)))

  await page.goto('/web/harness.html')
  await page.addScriptTag({ url: '/dist/wasm_exec.js' })

  const result = await page.evaluate(async () => {
    const { createFountain } = await import('/dist/fountain.mjs')
    if (typeof globalThis.Go !== 'function') return { injected: false }
    const fountain = await createFountain({ injectWasmExec: false })
    return {
      injected: true,
      title: fountain.parse('Title: T\n\nEXT. CASA - DIA\n\nTexto.\n').titlePage.title,
    }
  })

  expect(result).toEqual({ injected: true, title: 'T' })
  expect(errors).toEqual([])
})
