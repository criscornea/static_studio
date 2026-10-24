import { useMemo } from 'react'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { buildCommands } from '@/lib/commands'
import { formatShortcut } from '@/lib/shortcuts'
import { useUiStore } from '@/stores/ui'

/** Every keyboard shortcut, read from the command registry so the list cannot drift. */
export function ShortcutsDialog() {
  const open = useUiStore((s) => s.shortcutsOpen)
  const setOpen = useUiStore((s) => s.setShortcutsOpen)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Keyboard shortcuts</DialogTitle>
          <DialogDescription>Every command is also in the command palette.</DialogDescription>
        </DialogHeader>
        {open && <ShortcutList />}
      </DialogContent>
    </Dialog>
  )
}

function ShortcutList() {
  const rows = useMemo(() => buildCommands().filter((c) => c.shortcut !== undefined), [])

  return (
    <dl className="max-h-96 divide-y overflow-auto text-sm">
      {rows.map((c) => (
        <div key={c.id} className="flex items-center justify-between gap-4 py-1.5">
          <dt>{c.title}</dt>
          <dd>
            <kbd className="rounded border bg-muted px-1.5 py-0.5 text-xs">
              {formatShortcut(c.shortcut ?? '')}
            </kbd>
          </dd>
        </div>
      ))}
    </dl>
  )
}
