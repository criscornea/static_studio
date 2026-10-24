import { useEffect, useRef } from 'react'

import { Button } from '@/components/ui/button'
import { useBuildStore } from '@/stores/build'

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
      <pre
        ref={log}
        className="h-48 overflow-auto bg-muted px-4 py-2 font-mono text-xs whitespace-pre-wrap"
      >
        {lines.join('\n')}
      </pre>
    </section>
  )
}
