import { useEffect, useState } from 'react'

import { api } from '@/api/client'
import { asApiError } from '@/api/types'
import { Eye, MonitorOff } from 'lucide-react'

import { EmptyState, SkeletonRows } from '@/components/EmptyState'
import { IconButton } from '@/components/IconButton'
import { cn } from '@/lib/utils'
import { useDevServerStore } from '@/stores/devserver'

/** The status bar item that shows the preview server state and starts or stops it. */
export function DevServerStatus() {
  const running = useDevServerStore((s) => s.running)
  const ready = useDevServerStore((s) => s.ready)
  const busy = useDevServerStore((s) => s.busy)
  const error = useDevServerStore((s) => s.error)
  const url = useDevServerStore((s) => s.url)
  const output = useDevServerStore((s) => s.output)
  const refresh = useDevServerStore((s) => s.refresh)
  const start = useDevServerStore((s) => s.start)
  const stop = useDevServerStore((s) => s.stop)

  // The server outlives a page reload, so pick up its state.
  useEffect(() => {
    void refresh()
  }, [refresh])

  const port = url ? new URL(url).port : ''
  let text = 'Server: stopped'
  if (error) {
    text = 'Server: error'
  } else if (running) {
    text = ready ? `Server: running${port ? ` :${port}` : ''}` : 'Server: starting'
  }
  const tip = running ? 'Stop preview server' : 'Start preview server'

  return (
    <button
      type="button"
      title={error ? `${error}\n${output.join('\n')}`.trim() : tip}
      aria-label={tip}
      disabled={busy}
      onClick={() => void (running ? stop() : start())}
      className={cn(
        'rounded px-1.5 hover:bg-accent hover:text-accent-foreground disabled:opacity-50',
        error && 'text-destructive',
      )}
    >
      {text}
    </button>
  )
}

/** Toggles the preview next to the editor. Needs a ready preview server. */
export function PreviewButton({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  const ready = useDevServerStore((s) => s.ready)

  return (
    <IconButton
      label={open ? 'Hide preview' : 'Preview'}
      tip={ready ? undefined : 'Start the preview server first'}
      variant={open ? 'secondary' : 'ghost'}
      disabled={!ready}
      onClick={onToggle}
    >
      <Eye />
    </IconButton>
  )
}

/**
 * The Hugo-rendered page. The iframe loads straight from the dev server's own
 * origin (another port), so the site's scripts cannot reach this app's /api.
 */
export function PreviewFrame({ path }: { path: string }) {
  const ready = useDevServerStore((s) => s.ready)
  const running = useDevServerStore((s) => s.running)
  const busy = useDevServerStore((s) => s.busy)
  const start = useDevServerStore((s) => s.start)
  // Keyed by path so a result always belongs to the page it was asked for.
  const [result, setResult] = useState<{ url?: string; error?: string } | null>(null)

  useEffect(() => {
    if (!ready) {
      return
    }
    let cancelled = false
    api
      .devServer(path)
      .then((st) => {
        if (!cancelled) {
          setResult(
            st.pageUrl ? { url: st.pageUrl } : { error: 'This page has no preview address.' },
          )
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setResult({ error: asApiError(err).message })
        }
      })
    return () => {
      cancelled = true
    }
  }, [path, ready])

  if (!ready) {
    if (running) {
      return <SkeletonRows label="Starting the preview server" className="p-4" />
    }
    return (
      <EmptyState
        icon={MonitorOff}
        className="h-full"
        text="The preview server is not running."
        action={busy ? undefined : { label: 'Start server', onClick: () => void start() }}
      />
    )
  }
  if (result?.error) {
    return <p className="p-3 text-sm text-destructive">{result.error}</p>
  }
  if (!result?.url) {
    return <SkeletonRows label="Loading preview..." className="p-4" />
  }

  return (
    <iframe
      title="Page preview"
      src={result.url}
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
      className="block h-full w-full border-0 bg-white"
    />
  )
}
