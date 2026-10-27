import { ChevronRight, Save } from 'lucide-react'

import { BuildButton } from '@/components/BuildPanel'
import { PreviewButton } from '@/components/DevServer'
import { IconButton } from '@/components/IconButton'
import { PublishButton } from '@/components/PublishDialog'
import { useLayoutStore } from '@/stores/layout'
import { usePageStore } from '@/stores/page'

/** Path segments shown as a breadcrumb, e.g. content / posts / a.md. */
function pathSegments(path: string): string[] {
  return path.split('/').filter(Boolean)
}

/** The strip above the editor: where the page lives and whether it is saved. */
export function EditorHeader() {
  const path = usePageStore((s) => s.path)
  const page = usePageStore((s) => s.page)
  const dirty = usePageStore((s) => s.dirty)
  const savingPath = usePageStore((s) => s.savingPath)
  const saveError = usePageStore((s) => s.saveError)
  const save = usePageStore((s) => s.save)
  const previewOpen = useLayoutStore((s) => s.previewOpen)
  const togglePreview = useLayoutStore((s) => s.togglePreview)

  const saving = page !== null && savingPath === page.path
  const conflict = saveError?.code === 'page_conflict'
  const status = saving ? 'Saving…' : conflict ? 'Conflict' : dirty ? 'Unsaved changes' : 'Saved'
  const segments = path === null ? [] : pathSegments(path)

  return (
    <div className="flex h-9 shrink-0 items-center gap-2 border-b px-3">
      <nav aria-label="Breadcrumb" className="min-w-0 flex-1">
        <ol className="flex items-center gap-1 truncate font-mono text-xs text-muted-foreground">
          {segments.map((seg, i) => (
            <li key={i} className="flex items-center gap-1">
              {i > 0 && <ChevronRight className="size-3" aria-hidden />}
              <span className={i === segments.length - 1 ? 'text-foreground' : undefined}>
                {seg}
              </span>
            </li>
          ))}
        </ol>
      </nav>
      <PreviewButton open={previewOpen} onToggle={togglePreview} />
      <span className="mx-1 h-4 w-px bg-border" aria-hidden />
      {page !== null && (
        <IconButton
          label="Save"
          tip={`Save (${status})`}
          shortcut="Ctrl/Cmd+S"
          onClick={() => void save()}
          disabled={saving || !dirty}
          className={conflict ? 'text-destructive' : dirty ? 'text-foreground' : undefined}
        >
          <Save />
        </IconButton>
      )}
      <BuildButton />
      <PublishButton />
    </div>
  )
}
