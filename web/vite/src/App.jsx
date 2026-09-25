import { useEffect, useState } from 'react'
import { loadFountain } from './fountain.js'

const SAMPLE = `Title: LA SEÑAL
Author: Jane Doe

INT. HOUSE

Someone waits.
`

export default function App() {
  const [fountain, setFountain] = useState(null)
  const [error, setError] = useState(null)
  const [text, setText] = useState(SAMPLE)
  const [result, setResult] = useState({ elements: 0, warnings: [] })
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let alive = true
    loadFountain().then(
      (f) => alive && setFountain(f),
      (err) => alive && setError(String(err?.message ?? err)),
    )
    return () => {
      alive = false
    }
  }, [])

  // parse and lint are synchronous wasm calls, so they block the main thread.
  // rAF collapses a burst of keystrokes into one parse per frame, which is what
  // keeps typing smooth on a long script.
  useEffect(() => {
    if (!fountain) return
    let queued = false
    const frame = () => {
      queued = false
      try {
        const doc = fountain.parse(text)
        setResult({ elements: doc.elements.length, warnings: fountain.lint(text) })
        setError(null)
      } catch (err) {
        setError(String(err?.message ?? err))
      }
    }
    const schedule = () => {
      if (!queued) {
        queued = true
        requestAnimationFrame(frame)
      }
    }
    schedule()
    return () => cancelAnimationFrame(0)
  }, [fountain, text])

  const download = async () => {
    setBusy(true)
    try {
      const bytes = await fountain.renderPDF(text)
      const url = URL.createObjectURL(new Blob([bytes], { type: 'application/pdf' }))
      const a = document.createElement('a')
      a.href = url
      a.download = 'script.pdf'
      a.click()
      URL.revokeObjectURL(url)
      setError(null)
    } catch (err) {
      setError(String(err?.message ?? err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <h1>fountain + vite</h1>
      <p id="status" className={error ? 'error' : ''}>
        {error ?? (fountain ? 'ready' : 'booting…')}
      </p>

      <textarea value={text} onChange={(e) => setText(e.target.value)} spellCheck={false} />

      <div>
        <button onClick={download} disabled={!fountain || busy}>
          {busy ? 'loading renderer…' : 'Download PDF'}
        </button>
      </div>

      <p id="count">{result.elements} elements</p>

      <ul id="warnings">
        {result.warnings.map((w, i) => (
          <li key={i}>
            line {w.line} — <code>{w.code}</code> {w.message}
          </li>
        ))}
      </ul>
    </div>
  )
}
