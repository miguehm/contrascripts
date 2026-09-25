import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The wasm package needs no special handling here. `make package` stages the
// two .wasm files, wasm_exec.js and the wrapper into public/fountain, the
// assets are served from there untouched, and the app passes
// `base: '/fountain'` — see src/fountain.js.
//
// One thing is worth knowing if you set this up yourself: a Vite app cannot
// import a .mjs out of public/. It refuses the import outright, and the
// `?url` escape hatch it suggests answers with the SPA index.html and a 200.
// Keeping the wrapper in source and the heavy files in public/ avoids it.
export default defineConfig({
  plugins: [react()],
  // Bound to the IPv4 loopback explicitly. Vite's default is "localhost",
  // which resolves to ::1 first, and a tool waiting on 127.0.0.1 would then
  // never see the server come up.
  server: { host: '127.0.0.1', port: 5173 },
  preview: { host: '127.0.0.1', port: 4174 },
})
