// The one piece of wiring every frontend needs, and the piece most likely to be
// gotten wrong: boot the wasm once and share it.
//
// createFountain() is async and starts a Go runtime, so calling it from a
// component body or effect means a new runtime per mount. React StrictMode
// double-invokes effects in development and Vite re-evaluates modules on HMR,
// so both would leak a runtime into globalThis on every change.
//
// A module-level variable is not enough: HMR re-runs the module and resets it.
// A Symbol.for key on globalThis survives both, and is the same symbol across
// realms, so it stays a single instance.
//
// Path note: this file is web/vite/src/, so the fountain repo's dist/ is three
// levels up, not two. Only the wrapper is imported from there — the assets were
// staged into public/fountain by `make package` and are fetched at their own
// URLs, which is why they can be served without the bundler touching them.
import { createFountain } from '../../../dist/fountain.mjs'

const INSTANCE = Symbol.for('fountain.instance')

export function loadFountain() {
  globalThis[INSTANCE] ??= boot()
  return globalThis[INSTANCE]
}

async function boot() {
  // Exposed so the smoke test can assert a single runtime. A naive
  // createFountain() inside an effect lands here twice under StrictMode.
  window.__fountainBoots = (window.__fountainBoots ?? 0) + 1

  // `base` is required here. This module is compiled into the app bundle, so
  // fountain.mjs's own import.meta.url becomes the output chunk and is no
  // longer a sibling of the .wasm files. Both modules and the shim are resolved
  // from `base`, which is where `make package` staged them.
  return createFountain({ base: '/fountain' })
}
