import { useState } from 'react'
import { ChevronRight, Pencil, Trash2 } from 'lucide-react'

import type { ContentNode } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { formatDate } from '@/lib/format'

interface ContentTreeProps {
  node: ContentNode
  selected: string | null
  onSelect: (path: string) => void
  // When given, file rows show a rename or delete button.
  onRename?: (path: string) => void
  onDelete?: (path: string) => void
  // Paths of pages the last build reported errors for.
  errorPaths?: ReadonlySet<string>
  // Nesting level, used for indentation. The root renders its children at 0.
  depth?: number
}

export function ContentTree({ node, selected, onSelect, onRename, onDelete, errorPaths, depth = 0 }: ContentTreeProps) {
  // Each folder instance tracks its own state. Folders start expanded.
  const [expanded, setExpanded] = useState(true)
  const indent = { paddingLeft: `${depth * 0.75 + 0.5}rem` }

  if (!node.isDir) {
    const title = node.fields?.title
    const date = formatDate(node.fields?.date, { dateStyle: 'medium' })

    return (
      <li className="group relative">
        <button
          type="button"
          style={indent}
          onClick={() => onSelect(node.path)}
          className={cn(
            'block w-full rounded-md py-1 pr-2 text-left text-sm hover:bg-accent',
            selected === node.path && 'bg-accent text-accent-foreground',
          )}
        >
          <span className="flex items-center gap-2">
            <span className="min-w-0 flex-1 truncate">{title || node.name}</span>
            {errorPaths?.has(node.path) && (
              <Badge variant="destructive" className="shrink-0" title="The last build reported an error here">
                Error
              </Badge>
            )}
            {node.fields?.draft && (
              <Badge variant="secondary" className="shrink-0">
                Draft
              </Badge>
            )}
          </span>
          {(title || date) && (
            <span className="block truncate text-xs text-muted-foreground">
              {[title ? node.name : null, date].filter(Boolean).join(' · ')}
            </span>
          )}
        </button>
        {(onRename || onDelete) && (
          <span className="absolute top-1 right-1 hidden gap-0.5 rounded-md bg-background/90 group-focus-within:flex group-hover:flex">
            {onRename && (
              <button
                type="button"
                aria-label={`Rename ${node.name}`}
                title="Rename or move"
                onClick={() => onRename(node.path)}
                className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
              >
                <Pencil className="size-3.5" />
              </button>
            )}
            {onDelete && (
              <button
                type="button"
                aria-label={`Delete ${node.name}`}
                title="Delete"
                onClick={() => onDelete(node.path)}
                className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-destructive"
              >
                <Trash2 className="size-3.5" />
              </button>
            )}
          </span>
        )}
      </li>
    )
  }

  return (
    <li>
      <button
        type="button"
        aria-expanded={expanded}
        style={indent}
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-1 py-1 text-left text-[0.8125rem] font-semibold text-muted-foreground"
      >
        <ChevronRight className={cn('size-3 transition-transform', expanded && 'rotate-90')} />
        {node.name}
      </button>
      {expanded && (
        <ul>
          {node.children?.map((child) => (
            <ContentTree
              key={child.path}
              node={child}
              selected={selected}
              onSelect={onSelect}
              onRename={onRename}
              onDelete={onDelete}
              errorPaths={errorPaths}
              depth={depth + 1}
            />
          ))}
        </ul>
      )}
    </li>
  )
}
