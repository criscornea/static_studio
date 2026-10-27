import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type FocusEvent,
  type KeyboardEvent,
} from 'react'
import {
  ChevronRight,
  CircleAlert,
  FilePlus,
  FileText,
  Folder,
  FolderOpen,
  Pencil,
  Trash2,
  Copy,
  ExternalLink,
} from 'lucide-react'

import type { ContentNode } from '@/api/types'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from '@/components/ui/context-menu'
import { cn } from '@/lib/utils'
import { useViewStore } from '@/stores/view'

interface TreeFocus {
  // Path of the row that is in the tab order. Null means the first row.
  focusPath: string | null
  setFocusPath: (path: string) => void
}

// Roving focus state shared by every row of one tree.
const TreeFocusContext = createContext<TreeFocus | null>(null)

/** The `role="tree"` container. It owns the roving focus state of its rows. */
export function TreeRoot({ children, label }: { children: React.ReactNode; label: string }) {
  const [focusPath, setFocusPath] = useState<string | null>(null)
  const ref = useRef<HTMLUListElement>(null)

  // Forget a focus target that left the tree (filtered out, deleted).
  useEffect(() => {
    if (
      focusPath !== null &&
      !ref.current?.querySelector(`[data-path="${CSS.escape(focusPath)}"]`)
    ) {
      setFocusPath(null)
    }
  }, [focusPath, children])

  // Until a row has focus the tree itself is the tab stop and hands focus to the first row.
  function onFocus(e: FocusEvent<HTMLUListElement>) {
    if (e.target === e.currentTarget) {
      e.currentTarget.querySelector<HTMLElement>('[role="treeitem"]')?.focus()
    }
  }

  return (
    <TreeFocusContext.Provider value={{ focusPath, setFocusPath }}>
      <ul
        ref={ref}
        role="tree"
        aria-label={label}
        tabIndex={focusPath === null ? 0 : -1}
        onFocus={onFocus}
        className="outline-none"
      >
        {children}
      </ul>
    </TreeFocusContext.Provider>
  )
}

interface ContentTreeProps {
  node: ContentNode
  selected: string | null
  onSelect: (path: string) => void
  // Context menu actions. A menu entry shows only when its handler is given.
  onRename?: (path: string) => void
  onDelete?: (path: string) => void
  // Called with the node whose folder should receive the new page.
  onNewPage?: (node: ContentNode) => void
  // Paths of pages the last build reported errors for.
  errorPaths?: ReadonlySet<string>
  // Nesting level, used for indentation. The root renders its children at 0.
  depth?: number
}

const INDENT = 12

function visibleRows(row: HTMLElement): HTMLElement[] {
  const tree = row.closest('[role="tree"]') ?? row.parentElement
  if (!tree) {
    return []
  }
  return Array.from(tree.querySelectorAll<HTMLElement>('[role="treeitem"]')).filter(
    (el) => !el.closest('[hidden]'),
  )
}

export function ContentTree({
  node,
  selected,
  onSelect,
  onRename,
  onDelete,
  onNewPage,
  errorPaths,
  depth = 0,
}: ContentTreeProps) {
  // Each folder instance tracks its own state. Folders start expanded.
  const [expanded, setExpanded] = useState(true)
  const focus = useContext(TreeFocusContext)
  const collapseEpoch = useViewStore((s) => s.collapseEpoch)
  const seenEpoch = useRef(collapseEpoch)
  const rowRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (seenEpoch.current !== collapseEpoch) {
      seenEpoch.current = collapseEpoch
      setExpanded(false)
    }
  }, [collapseEpoch])

  const title = node.fields?.title
  const label = node.isDir ? node.name : title || node.name
  const tabIndex = focus === null || focus.focusPath === node.path ? 0 : -1

  function move(to: HTMLElement | undefined) {
    to?.focus()
  }

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const row = e.currentTarget
    const rows = visibleRows(row)
    const index = rows.indexOf(row)
    switch (e.key) {
      case 'ArrowDown':
        move(rows[index + 1])
        break
      case 'ArrowUp':
        move(rows[index - 1])
        break
      case 'Home':
        move(rows[0])
        break
      case 'End':
        move(rows[rows.length - 1])
        break
      case 'ArrowRight':
        if (node.isDir) {
          if (!expanded) {
            setExpanded(true)
          } else {
            move(rows[index + 1])
          }
        }
        break
      case 'ArrowLeft':
        if (node.isDir && expanded) {
          setExpanded(false)
        } else {
          const group = row.closest('[role="group"]')
          move(
            group?.parentElement?.querySelector<HTMLElement>(':scope > [role="treeitem"]') ??
              undefined,
          )
        }
        break
      case 'Enter':
      case ' ':
        if (node.isDir) {
          setExpanded(!expanded)
        } else {
          onSelect(node.path)
        }
        break
      default:
        return
    }
    e.preventDefault()
  }

  function activate() {
    if (node.isDir) {
      setExpanded(!expanded)
    } else {
      onSelect(node.path)
    }
  }

  const Icon = node.isDir ? (expanded ? FolderOpen : Folder) : FileText
  const isSelected = !node.isDir && selected === node.path

  return (
    <li role="none">
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <div
            ref={rowRef}
            role="treeitem"
            aria-level={depth + 1}
            aria-expanded={node.isDir ? expanded : undefined}
            aria-selected={node.isDir ? undefined : isSelected}
            tabIndex={tabIndex}
            title={node.path}
            data-path={node.path}
            onClick={activate}
            onFocus={() => focus?.setFocusPath(node.path)}
            onKeyDown={onKeyDown}
            style={{ paddingLeft: `${depth * INDENT + 6}px` }}
            className={cn(
              'relative flex h-[1.4375rem] cursor-default items-center gap-1 pr-2 text-[0.8125rem] outline-none select-none hover:bg-accent focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-inset',
              node.isDir && 'text-muted-foreground',
              isSelected && 'bg-accent font-medium text-accent-foreground',
            )}
          >
            {Array.from({ length: depth }, (_, i) => (
              <span
                key={i}
                aria-hidden
                className="absolute inset-y-0 w-px bg-border"
                style={{ left: `${i * INDENT + 12}px` }}
              />
            ))}
            {node.isDir ? (
              <ChevronRight
                className={cn('size-3 shrink-0 transition-transform', expanded && 'rotate-90')}
              />
            ) : (
              <span className="w-3 shrink-0" />
            )}
            <Icon
              className={cn(
                'size-3.5 shrink-0',
                !node.isDir && node.fields?.draft && 'text-yellow-500',
              )}
            />
            <span className="min-w-0 flex-1 truncate">{label}</span>
            {!node.isDir && errorPaths?.has(node.path) && (
              <span
                className="shrink-0 text-destructive"
                title="The last build reported an error here"
              >
                <CircleAlert className="size-3.5" />
                <span className="sr-only">Error</span>
              </span>
            )}
          </div>
        </ContextMenuTrigger>
        <ContextMenuContent>
          {!node.isDir && (
            <ContextMenuItem onSelect={() => onSelect(node.path)}>
              <ExternalLink /> Open
            </ContextMenuItem>
          )}
          {onNewPage && (
            <ContextMenuItem onSelect={() => onNewPage(node)}>
              <FilePlus /> New page here
            </ContextMenuItem>
          )}
          {!node.isDir && onRename && (
            <ContextMenuItem onSelect={() => onRename(node.path)}>
              <Pencil /> Rename
            </ContextMenuItem>
          )}
          {!node.isDir && onDelete && (
            <ContextMenuItem variant="destructive" onSelect={() => onDelete(node.path)}>
              <Trash2 /> Delete
            </ContextMenuItem>
          )}
          <ContextMenuSeparator />
          <ContextMenuItem onSelect={() => void navigator.clipboard?.writeText(node.path)}>
            <Copy /> Copy path
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
      {node.isDir && (
        <ul role="group" hidden={!expanded}>
          {node.children?.map((child) => (
            <ContentTree
              key={child.path}
              node={child}
              selected={selected}
              onSelect={onSelect}
              onRename={onRename}
              onDelete={onDelete}
              onNewPage={onNewPage}
              errorPaths={errorPaths}
              depth={depth + 1}
            />
          ))}
        </ul>
      )}
    </li>
  )
}
