import { useEffect, useMemo, useRef, useState } from 'react'
import { ChevronsDownUp, ListFilter, Plus, Search, X } from 'lucide-react'

import { ContentTree, TreeRoot } from '@/components/ContentTree'
import { DeletePageDialog } from '@/components/DeletePageDialog'
import { NewPageDialog } from '@/components/NewPageDialog'
import { RenamePageDialog } from '@/components/RenamePageDialog'
import { SearchResults } from '@/components/SearchResults'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  applyView,
  collectTags,
  folderOf,
  isDefaultView,
  type SortKey,
  type StatusFilter,
} from '@/lib/tree'
import { useBuildStore } from '@/stores/build'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import { useSearchStore } from '@/stores/search'
import { useViewStore } from '@/stores/view'

const sortLabels: Record<SortKey, string> = {
  name: 'Name',
  title: 'Title',
  'date-desc': 'Newest first',
  'date-asc': 'Oldest first',
}

const statusLabels: Record<StatusFilter, string> = {
  all: 'All pages',
  draft: 'Drafts',
  published: 'Published',
}

// Radix radio items cannot use an empty string as a value.
const allTags = '__all__'

function IconButton({
  label,
  onClick,
  children,
}: {
  label: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon-sm" aria-label={label} onClick={onClick}>
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}

export function Sidebar() {
  const tree = useProjectStore((s) => s.tree)
  const selected = usePageStore((s) => s.path)
  const problems = useBuildStore((s) => s.problems)
  const errorPaths = useMemo(() => new Set(problems.map((p) => p.file)), [problems])
  const load = usePageStore((s) => s.load)

  const search = useSearchStore()
  const view = useViewStore()

  const searchBox = useRef<HTMLInputElement>(null)
  const contentDir = useProjectStore((s) => s.project?.contentDir ?? 'content')
  const [creating, setCreating] = useState(false)
  // Undefined lets the dialog pick the open page's folder.
  const [newFolder, setNewFolder] = useState<string | undefined>()
  const [renaming, setRenaming] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)
  const searching = search.query.trim() !== ''

  // Cmd/Ctrl+K jumps to the search box from anywhere, like a native app.
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        searchBox.current?.focus()
        searchBox.current?.select()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  function openNewPage(folder: string | undefined) {
    setNewFolder(folder === '' ? undefined : folder)
    setCreating(true)
  }

  const nodes = useMemo(() => tree?.children ?? [], [tree])
  const tags = useMemo(() => collectTags(nodes), [nodes])
  const { sort, status, tag } = view
  const visible = useMemo(() => applyView(nodes, { sort, status, tag }), [nodes, sort, status, tag])
  const filtered = !isDefaultView({ sort, status, tag })

  return (
    <TooltipProvider>
      <div className="flex h-full min-h-0 flex-col bg-sidebar text-sidebar-foreground">
        <div className="space-y-1.5 border-b p-2">
          <div className="flex items-center gap-0.5">
            <h2 className="flex-1 truncate px-1 text-[0.6875rem] font-semibold tracking-wide text-muted-foreground uppercase">
              Pages
            </h2>
            <IconButton label="New page" onClick={() => openNewPage('')}>
              <Plus />
            </IconButton>
            <DropdownMenu>
              <Tooltip>
                <TooltipTrigger asChild>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Sort and filter"
                      className="relative"
                    >
                      <ListFilter />
                      {filtered && (
                        <span className="absolute top-1 right-1 size-1.5 rounded-full bg-primary" />
                      )}
                    </Button>
                  </DropdownMenuTrigger>
                </TooltipTrigger>
                <TooltipContent>Sort and filter</TooltipContent>
              </Tooltip>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel>Sort by</DropdownMenuLabel>
                <DropdownMenuRadioGroup
                  value={sort}
                  onValueChange={(v) => view.setSort(v as SortKey)}
                >
                  {Object.entries(sortLabels).map(([value, label]) => (
                    <DropdownMenuRadioItem key={value} value={value}>
                      {label}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
                <DropdownMenuSeparator />
                <DropdownMenuLabel>Show</DropdownMenuLabel>
                <DropdownMenuRadioGroup
                  value={status}
                  onValueChange={(v) => view.setStatus(v as StatusFilter)}
                >
                  {Object.entries(statusLabels).map(([value, label]) => (
                    <DropdownMenuRadioItem key={value} value={value}>
                      {label}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
                {tags.length > 0 && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuLabel>Tag</DropdownMenuLabel>
                    <DropdownMenuRadioGroup
                      value={tag ?? allTags}
                      onValueChange={(v) => view.setTag(v === allTags ? null : v)}
                    >
                      <DropdownMenuRadioItem value={allTags}>All tags</DropdownMenuRadioItem>
                      {tags.map((t) => (
                        <DropdownMenuRadioItem key={t} value={t}>
                          {t}
                        </DropdownMenuRadioItem>
                      ))}
                    </DropdownMenuRadioGroup>
                  </>
                )}
                {filtered && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onSelect={view.reset}>
                      Reset sort and filters
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
            <IconButton label="Collapse all folders" onClick={view.collapseAll}>
              <ChevronsDownUp />
            </IconButton>
          </div>

          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={searchBox}
              type="search"
              value={search.query}
              onChange={(e) => search.setQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Escape' && search.clear()}
              placeholder="Search  (Ctrl+K)"
              aria-label="Search pages"
              spellCheck={false}
              autoComplete="off"
              className="h-7 px-7 text-[0.8125rem] [&::-webkit-search-cancel-button]:hidden"
            />
            {searching && (
              <button
                type="button"
                aria-label="Clear search"
                onClick={search.clear}
                className="absolute top-1/2 right-2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                <X className="size-3.5" />
              </button>
            )}
          </div>
        </div>

        <ScrollArea className="min-h-0 flex-1">
          <nav className="py-1">
            {searching ? (
              <SearchResults
                query={search.query}
                results={search.results}
                loading={search.loading}
                error={search.error}
                selected={selected}
                onSelect={(path) => void load(path)}
              />
            ) : visible.length > 0 ? (
              <TreeRoot label="Pages">
                {visible.map((child) => (
                  <ContentTree
                    key={child.path}
                    node={child}
                    selected={selected}
                    onSelect={(path) => void load(path)}
                    onRename={setRenaming}
                    onDelete={setDeleting}
                    onNewPage={(node) =>
                      openNewPage(folderOf(node.isDir ? `${node.path}/x` : node.path, contentDir))
                    }
                    errorPaths={errorPaths}
                  />
                ))}
              </TreeRoot>
            ) : (
              <div className="space-y-2 px-3 py-2 text-sm text-muted-foreground">
                <p>{filtered ? 'No pages match this filter.' : 'No content files yet.'}</p>
                {filtered && (
                  <Button variant="outline" size="sm" onClick={view.reset}>
                    Show all pages
                  </Button>
                )}
              </div>
            )}
          </nav>
        </ScrollArea>

        <NewPageDialog open={creating} onOpenChange={setCreating} folder={newFolder} />
        <RenamePageDialog path={renaming} onClose={() => setRenaming(null)} />
        <DeletePageDialog path={deleting} onClose={() => setDeleting(null)} />
      </div>
    </TooltipProvider>
  )
}
