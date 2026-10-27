import { Terminal } from 'lucide-react'
import { DevServerStatus } from '@/components/DevServer'
import { cn } from '@/lib/utils'
import { useLayoutStore } from '@/stores/layout'
import { useBuildStore, type BuildStatus } from '@/stores/build'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'

const buildText: Record<BuildStatus, string> = {
  idle: 'Build: idle',
  running: 'Build: running',
  succeeded: 'Build: succeeded',
  failed: 'Build: failed',
}

function baseName(root: string): string {
  const parts = root.split(/[\\/]/).filter(Boolean)
  return parts[parts.length - 1] ?? root
}

/** The bottom bar: project, save, server and build state. */
export function StatusBar() {
  const root = useProjectStore((s) => s.project?.root)
  const page = usePageStore((s) => s.page)
  const dirty = usePageStore((s) => s.dirty)
  const savingPath = usePageStore((s) => s.savingPath)
  const saveError = usePageStore((s) => s.saveError)
  const build = useBuildStore((s) => s.status)
  const logOpen = useLayoutStore((s) => s.buildLogOpen)
  const toggleLog = useLayoutStore((s) => s.toggleBuildLog)

  const saving = page !== null && savingPath === page.path
  let save = ''
  if (page !== null) {
    if (saving) {
      save = 'Saving…'
    } else if (saveError?.code === 'page_conflict') {
      save = 'Conflict'
    } else if (saveError) {
      save = 'Save failed'
    } else {
      save = dirty ? 'Unsaved changes' : 'Saved'
    }
  }

  return (
    <footer
      aria-label="Status bar"
      className="col-span-2 flex items-center gap-4 border-t bg-sidebar px-3 text-xs text-muted-foreground"
    >
      <span className="truncate font-medium text-foreground" title={root}>
        {root ? baseName(root) : ''}
      </span>
      <span className="flex-1 truncate">{save}</span>
      <DevServerStatus />
      <button
        type="button"
        aria-pressed={logOpen}
        title="Toggle build log (Ctrl+J)"
        onClick={toggleLog}
        className={cn(
          'inline-flex items-center gap-1 rounded px-1.5 hover:bg-accent hover:text-accent-foreground',
          logOpen && 'bg-accent text-accent-foreground',
          build === 'failed' && 'text-destructive',
        )}
      >
        <Terminal aria-hidden="true" className="size-3.5" />
        {buildText[build]}
      </button>
    </footer>
  )
}
