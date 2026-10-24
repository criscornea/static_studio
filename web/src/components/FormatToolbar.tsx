import {
  Bold,
  Code,
  Heading1,
  Heading2,
  Heading3,
  Italic,
  Link,
  List,
  ListOrdered,
  Quote,
  Strikethrough,
  type LucideIcon,
} from 'lucide-react'

import { IconButton } from '@/components/IconButton'
import { formatShortcuts, formatTitles } from '@/lib/editorFormat'
import type { FormatId } from '@/lib/formatState'
import { formatShortcut } from '@/lib/shortcuts'
import { cn } from '@/lib/utils'
import { useEditorStore } from '@/stores/editor'

const groups: { id: FormatId; icon: LucideIcon }[][] = [
  [
    { id: 'h1', icon: Heading1 },
    { id: 'h2', icon: Heading2 },
    { id: 'h3', icon: Heading3 },
  ],
  [
    { id: 'bold', icon: Bold },
    { id: 'italic', icon: Italic },
    { id: 'strike', icon: Strikethrough },
    { id: 'code', icon: Code },
    { id: 'link', icon: Link },
  ],
  [
    { id: 'bullet', icon: List },
    { id: 'numbered', icon: ListOrdered },
    { id: 'quote', icon: Quote },
  ],
]

/** Floating formatting bar, centred at the bottom of the editor while it has focus. */
export function FormatToolbar() {
  const focused = useEditorStore((s) => s.focused)
  const active = useEditorStore((s) => s.active)
  const format = useEditorStore((s) => s.format)
  const setFocused = useEditorStore((s) => s.setFocused)

  if (!focused) return null

  return (
    <div
      role="toolbar"
      aria-label="Formatting"
      data-format-toolbar
      // Keeps the editor focused (and its selection) when a button is pressed.
      onMouseDown={(e) => e.preventDefault()}
      onFocus={() => setFocused(true)}
      onBlur={(e) => {
        const next = e.relatedTarget
        const view = useEditorStore.getState().view
        if (next instanceof Node && (e.currentTarget.contains(next) || view?.dom.contains(next))) {
          return
        }
        setFocused(false)
      }}
      className="absolute bottom-4 left-1/2 z-10 flex max-w-[calc(100%-1rem)] -translate-x-1/2 items-center gap-0.5 overflow-x-auto rounded-lg border bg-popover p-1 text-popover-foreground shadow-md"
    >
      {groups.map((group, i) => (
        <div key={i} className="flex items-center gap-0.5">
          {i > 0 && <span className="mx-1 h-5 w-px bg-border" aria-hidden />}
          {group.map(({ id, icon: Icon }) => {
            const on = active.includes(id)
            const key = formatShortcuts[id]
            return (
              <IconButton
                key={id}
                label={formatTitles[id]}
                shortcut={key ? ` ${formatShortcut(key)}` : undefined}
                aria-pressed={on}
                onClick={() => format(id)}
                className={cn(on && 'bg-accent text-accent-foreground')}
              >
                <Icon />
              </IconButton>
            )
          })}
        </div>
      ))}
    </div>
  )
}
