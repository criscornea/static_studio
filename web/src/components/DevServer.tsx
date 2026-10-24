import { useEffect, useState } from 'react'

import { api } from '@/api/client'
import { asApiError } from '@/api/types'
import { Button } from '@/components/ui/button'
import { useDevServerStore } from '@/stores/devserver'

/** The header control that starts and stops the Hugo preview server. */
export function DevServerButton() {
  const running = useDevServerStore((s) => s.running)
  const ready = useDevServerStore((s) => s.ready)
  const busy = useDevServerStore((s) => s.busy)
  const error = useDevServerStore((s) => s.error)
  const output = useDevServerStore((s) => s.output)
  const refresh = useDevServerStore((s) => s.refresh)
  const start = useDevServerStore((s) => s.start)
  const stop = useDevServerStore((s) => s.stop)

  // The server outlives a page reload, so pick up its state.
  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <div className="flex items-center gap-2">
      {error && (
        <span role="alert" className="text-xs text-destructive">
          {error}
          {output.length > 0 && (
            <pre className="max-h-32 overflow-auto whitespace-pre-wrap">{output.join('\n')}</pre>
          )}
        </span>
      )}
      {running && (
        <span role="status" className="text-xs text-muted-foreground">
          {ready ? 'Preview server running' : 'Preview server starting...'}
        </span>
      )}
      <Button
        variant="outline"
        size="sm"
        disabled={busy}
        onClick={() => void (running ? stop() : start())}
      >
        {running ? 'Stop preview server' : 'Start preview server'}
      </Button>
    </div>
  )
}

/** Toggles the preview next to the editor. Needs a ready preview server. */
export function PreviewButton({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  const ready = useDevServerStore((s) => s.ready)

  return (
    <Button
      variant="outline"
      size="sm"
      disabled={!ready}
      title={ready ? undefined : 'Start the preview server first'}
      onClick={onToggle}
    >
      {open ? 'Hide preview' : 'Preview'}
    </Button>
  )
}

/**
 * The Hugo-rendered page. The iframe loads straight from the dev server's own
 * origin (another port), so the site's scripts cannot reach this app's /api.
 */
export function PreviewFrame({ path }: { path: string }) {
  const ready = useDevServerStore((s) => s.ready)
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
    return <p className="mb-3 text-sm text-muted-foreground">The preview server is not running.</p>
  }
  if (result?.error) {
    return <p className="mb-3 text-sm text-destructive">{result.error}</p>
  }
  if (!result?.url) {
    return <p className="mb-3 text-sm text-muted-foreground">Loading preview...</p>
  }

  return (
    <iframe
      title="Page preview"
      src={result.url}
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
      className="mb-3 h-[32rem] w-full rounded-md border bg-white"
    />
  )
}
