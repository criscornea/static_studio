import { useBuildStore, type BuildStatus } from '@/stores/build'
import { useDevServerStore } from '@/stores/devserver'
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

/** The bottom bar: project, save, preview-server and build state. */
export function StatusBar() {
  const root = useProjectStore((s) => s.project?.root)
  const page = usePageStore((s) => s.page)
  const dirty = usePageStore((s) => s.dirty)
  const savingPath = usePageStore((s) => s.savingPath)
  const saveError = usePageStore((s) => s.saveError)
  const running = useDevServerStore((s) => s.running)
  const ready = useDevServerStore((s) => s.ready)
  const serverError = useDevServerStore((s) => s.error)
  const build = useBuildStore((s) => s.status)

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

  let server = 'Preview: stopped'
  if (serverError) {
    server = 'Preview: error'
  } else if (running) {
    server = ready ? 'Preview: running' : 'Preview: starting'
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
      <span>{server}</span>
      <span className={build === 'failed' ? 'text-destructive' : undefined}>
        {buildText[build]}
      </span>
    </footer>
  )
}
