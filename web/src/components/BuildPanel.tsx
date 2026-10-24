import { useEffect, useRef } from 'react'

import { Button } from '@/components/ui/button'
import { useBuildStore } from '@/stores/build'
import { usePageStore } from '@/stores/page'

const statusText = {
  idle: '',
  running: 'Building...',
  succeeded: 'Build finished.',
  failed: 'Build failed.',
} as const

/** The header button that starts a build. */
export function BuildButton() {
  const status = useBuildStore((s) => s.status)
  const start = useBuildStore((s) => s.start)

  return (
    <Button size="sm" disabled={status === 'running'} onClick={() => void start()}>
      Build
    </Button>
  )
}

/** The live log of the latest build. Hidden until a build has started. */
export function BuildPanel() {
  const status = useBuildStore((s) => s.status)
  const lines = useBuildStore((s) => s.lines)
  const error = useBuildStore((s) => s.error)
  const problems = useBuildStore((s) => s.problems)
  const jumpTo = useBuildStore((s) => s.jumpTo)
  const load = usePageStore((s) => s.load)
  const cancel = useBuildStore((s) => s.cancel)
  const reset = useBuildStore((s) => s.reset)
  const log = useRef<HTMLPreElement>(null)

  useEffect(() => {
    const el = log.current
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  }, [lines])

  if (status === 'idle') {
    return null
  }

  return (
    <section aria-label="Build output" className="border-t">
      <div className="flex items-center gap-2 px-4 py-1.5">
        <span role="status" className="flex-1 text-sm">
          {statusText[status]}
          {error && <span className="ml-2 text-destructive">{error}</span>}
        </span>
        {status === 'running' ? (
          <Button variant="ghost" size="sm" onClick={() => void cancel()}>
            Cancel
          </Button>
        ) : (
          <Button variant="ghost" size="sm" onClick={reset}>
            Dismiss
          </Button>
        )}
      </div>
      {problems.length > 0 && (
        <ul aria-label="Build errors" className="max-h-32 overflow-auto px-4 pb-2 text-sm">
          {problems.map((p, i) => (
            <li key={i}>
              {p.file.startsWith('content/') ? (
                <button
                  type="button"
                  onClick={() => void load(p.file).then(() => jumpTo(p))}
                  className="text-left text-destructive hover:underline"
                >
                  {p.file}:{p.line}
                </button>
              ) : (
                <span className="text-destructive">
                  {p.file}:{p.line}
                </span>
              )}
              <span className="ml-2 text-muted-foreground">{p.message}</span>
            </li>
          ))}
        </ul>
      )}
      <pre
        ref={log}
        className="h-48 overflow-auto bg-muted px-4 py-2 font-mono text-xs whitespace-pre-wrap"
      >
        {lines.join('\n')}
      </pre>
    </section>
  )
}
