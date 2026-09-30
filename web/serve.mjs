// Minimal static file server for the browser smoke test.
//
// It serves the repository root rather than dist/ so the spec can reach the
// real packaged layout (fountain.mjs resolving its own .wasm siblings) and the
// .fountain fixtures at the same time.
//
//   node serve.mjs [--port 4173] [--root ..]
//
// It is deliberately not a dependency: the smoke test is the only thing that
// needs it, and adding a server framework for two file reads would be noise.

import { createServer } from 'node:http'
import { createReadStream } from 'node:fs'
import { stat } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { extname, join, normalize, resolve, sep } from 'node:path'

const args = process.argv.slice(2)
const flag = (name, fallback) => {
  const i = args.indexOf(`--${name}`)
  return i >= 0 && args[i + 1] ? args[i + 1] : fallback
}

const PORT = Number(flag('port', process.env.FOUNTAIN_PORT || 4173))
const HOST = flag('host', '127.0.0.1')
const ROOT = resolve(fileURLToPath(new URL(flag('root', '..'), import.meta.url)))

const TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.wasm': 'application/wasm',
  '.fountain': 'text/plain; charset=utf-8',
  '.txt': 'text/plain; charset=utf-8',
}

const server = createServer(async (req, res) => {
  const send = (code, body, type = 'text/plain; charset=utf-8') => {
    res.writeHead(code, { 'content-type': type, 'cache-control': 'no-store' })
    res.end(body)
  }

  const url = new URL(req.url, `http://${req.headers.host || 'localhost'}`)
  if (url.pathname === '/ready') return send(200, 'ok')

  // ROOT is the repository root, which has no index.html of its own, and it is
  // the URL this server prints on startup. Send it to the demo rather than
  // 404ing the first thing anyone opens.
  if (url.pathname === '/') {
    res.writeHead(302, { location: '/web/', 'cache-control': 'no-store' })
    return res.end()
  }

  // Resolve inside ROOT and refuse anything that escapes it.
  const decoded = decodeURIComponent(url.pathname)
  let target = join(ROOT, normalize(decoded))
  if (target !== ROOT && !target.startsWith(ROOT + sep)) return send(403, 'forbidden')

  let info
  try {
    info = await stat(target)
  } catch {
    return send(404, `not found: ${decoded}`)
  }

  // A directory is a path the user typed, not a file: serve its index.html the
  // way any static server does. This is what makes /web/ work, the path
  // index.html documents for the live preview.
  if (info.isDirectory()) {
    target = join(target, 'index.html')
    try {
      info = await stat(target)
    } catch {
      return send(404, `no index.html in ${decoded}`)
    }
  }

  res.writeHead(200, {
    'content-type': TYPES[extname(target)] || 'application/octet-stream',
    'content-length': info.size,
    'cache-control': 'no-store',
  })
  createReadStream(target).pipe(res)
})

server.listen(PORT, HOST, () => {
  process.stdout.write(`serving ${ROOT} at http://${HOST}:${PORT}\n`)
})
