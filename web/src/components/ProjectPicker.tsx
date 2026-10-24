import { useEffect, useState } from 'react'

import { api } from '@/api/client'
import type { RecentProject } from '@/api/types'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useProjectStore } from '@/stores/project'

export function ProjectPicker() {
  const open = useProjectStore((s) => s.open)
  const loading = useProjectStore((s) => s.loading)
  const error = useProjectStore((s) => s.error)
  const [path, setPath] = useState('')
  const [recent, setRecent] = useState<RecentProject[]>([])

  useEffect(() => {
    let cancelled = false
    api
      .recentProjects()
      .then((list) => {
        if (!cancelled) {
          setRecent(list)
        }
      })
      .catch(() => {
        // The list is a convenience; the path field still works without it.
      })
    return () => {
      cancelled = true
    }
  }, [])

  async function removeRecent(entry: RecentProject) {
    try {
      await api.removeRecent(entry.path)
      setRecent((list) => list.filter((e) => e.path !== entry.path))
    } catch {
      // Leave the entry in place if removal failed.
    }
  }

  const trimmed = path.trim()

  async function submit() {
    if (trimmed === '') {
      return
    }
    await open(trimmed)
  }

  return (
    <div className="flex h-full items-center justify-center overflow-auto bg-sidebar p-6">
      <section className="w-full max-w-[32rem] rounded-lg border bg-background p-6 shadow-sm">
        <h1 className="mb-1 text-xl font-semibold">Static Studio</h1>
        <p className="mb-5 text-muted-foreground">Open the folder that contains your website.</p>

        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            void submit()
          }}
        >
          <Input
            value={path}
            onChange={(e) => setPath(e.target.value)}
            placeholder="/path/to/your/site"
            spellCheck={false}
            autoComplete="off"
            disabled={loading}
            className="font-mono"
          />
          <Button type="submit" disabled={loading || trimmed === ''}>
            {loading ? 'Opening…' : 'Open'}
          </Button>
        </form>

        {error && (
          <Alert variant="destructive" className="mt-4">
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        )}

        {recent.length > 0 && (
          <div className="mt-6">
            <h2 className="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
              Recent projects
            </h2>
            <ul className="max-h-64 divide-y overflow-auto rounded-md border">
              {recent.map((entry) => (
                <li
                  key={entry.path}
                  className="flex items-center gap-2 px-3 py-1.5 hover:bg-accent/50"
                >
                  {entry.available ? (
                    <button
                      type="button"
                      className="min-w-0 flex-1 text-left"
                      disabled={loading}
                      onClick={() => void open(entry.path)}
                    >
                      <span className="block truncate font-medium">{entry.name}</span>
                      <span className="block truncate font-mono text-xs text-muted-foreground">
                        {entry.path}
                      </span>
                    </button>
                  ) : (
                    <div className="min-w-0 flex-1 opacity-50" aria-disabled="true">
                      <span className="block truncate font-medium">{entry.name}</span>
                      <span className="block truncate font-mono text-xs">{entry.path}</span>
                      <span className="block text-xs">Folder not found</span>
                    </div>
                  )}
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    aria-label={`Remove ${entry.name} from recent projects`}
                    onClick={() => void removeRecent(entry)}
                  >
                    Remove
                  </Button>
                </li>
              ))}
            </ul>
          </div>
        )}

        <p className="mt-6 text-xs text-muted-foreground">
          This is the folder with your <code>hugo.toml</code> in it.
        </p>
      </section>
    </div>
  )
}
