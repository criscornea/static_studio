import { useState } from 'react'
import { ChevronRight } from 'lucide-react'

import type { ContentNode } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { formatDate } from '@/lib/format'

interface ContentTreeProps {
  node: ContentNode
  selected: string | null
  onSelect: (path: string) => void
  // Nesting level, used for indentation. The root renders its children at 0.
  depth?: number
}

export function ContentTree({ node, selected, onSelect, depth = 0 }: ContentTreeProps) {
  // Each folder instance tracks its own state. Folders start expanded.
  const [expanded, setExpanded] = useState(true)
  const indent = { paddingLeft: `${depth * 0.75 + 0.5}rem` }

  if (!node.isDir) {
    const title = node.fields?.title
    const date = formatDate(node.fields?.date, { dateStyle: 'medium' })

    return (
      <li>
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
              depth={depth + 1}
            />
          ))}
        </ul>
      )}
    </li>
  )
}
