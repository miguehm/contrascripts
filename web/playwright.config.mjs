import { defineConfig } from '@playwright/test'

const PORT = Number(process.env.FOUNTAIN_PORT || 4173)
const BASE = `http://127.0.0.1:${PORT}`

// The spec is a single serial run on one page: the point of the exercise is
// that both modules live in the same document, so splitting it across parallel
// pages would hide the very failure it is meant to catch.
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: 'list',
  timeout: 60_000,
  webServer: {
    command: `node serve.mjs --port ${PORT}`,
    url: `${BASE}/ready`,
    reuseExistingServer: !process.env.CI,
    stdout: 'pipe',
  },
  use: { baseURL: BASE },
})
