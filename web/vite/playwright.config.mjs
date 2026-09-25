import { defineConfig } from '@playwright/test'

const DEV = 'http://127.0.0.1:5173'
const PREVIEW = 'http://127.0.0.1:4174'

// The fixture is exercised twice on purpose. `vite dev` serves the modules
// unbundled from public/, while `vite build` compiles the app and serves the
// bundle — a different path through the toolchain, and the one where
// import.meta.url stops pointing at the .wasm files. Passing here is not
// evidence about the other.
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  reporter: 'list',
  timeout: 60_000,
  webServer: [
    // Plain vite, no sync: the preview below stages public/ first, and running
    // `make package` from both at once would race on the same directory.
    { command: 'npx vite', url: DEV, reuseExistingServer: !process.env.CI, timeout: 120_000 },
    {
      command: 'npm run build && npx vite preview',
      url: PREVIEW,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
  projects: [
    { name: 'dev', use: { baseURL: DEV } },
    { name: 'preview', use: { baseURL: PREVIEW } },
  ],
})
