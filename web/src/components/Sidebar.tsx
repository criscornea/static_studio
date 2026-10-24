import { useEffect, useMemo, useRef } from 'react'
import { ChevronsDownUp, FilePlus, ListFilter, Plus, Search, X } from 'lucide-react'

import { ContentTree, TreeRoot } from '@/components/ContentTree'
import { SearchResults } from '@/components/SearchResults'
import { EmptyState, SkeletonRows } from '@/components/EmptyState'
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
import { useUiStore } from '@/stores/ui'
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
  const roots = useProjectStore((s) => s.roots)
  const selected = usePageStore((s) => s.path)
  const problems = useBuildStore((s) => s.problems)
  const errorPaths = useMemo(() => new Set(problems.map((p) => p.file)), [problems])
  const load = usePageStore((s) => s.load)

  const search = useSearchStore()
  const view = useViewStore()

  const searchBox = useRef<HTMLInputElement>(null)
  const contentDir = useProjectStore((s) => s.project?.contentDir ?? 'content')
  const openNewPage = useUiStore((s) => s.openNewPage)
  const openRename = useUiStore((s) => s.openRename)
  const openDelete = useUiStore((s) => s.openDelete)
  const focusRequested = useUiStore((s) => s.searchFocusRequested)
  const clearSearchFocus = useUiStore((s) => s.clearSearchFocus)
  const searching = search.query.trim() !== ''

  // The focus-search command (Ctrl/Cmd+K) lands here.
  useEffect(() => {
    if (focusRequested) {
      searchBox.current?.focus()
      searchBox.current?.select()
      clearSearchFocus()
    }
  }, [focusRequested, clearSearchFocus])

  // Every content dir gets its own labelled group; a single dir needs no label.
  const groups = useMemo(
    () =>
      roots.length > 0
        ? roots.map((r) => ({ dir: r.dir, lang: r.lang, children: r.root.children ?? [] }))
        : tree
          ? [{ dir: tree.path, lang: undefined, children: tree.children ?? [] }]
          : [],
    [roots, tree],
  )
  const labelled = groups.length > 1
  const nodes = useMemo(() => groups.flatMap((g) => g.children), [groups])
  const tags = useMemo(() => collectTags(nodes), [nodes])
  const { sort, status, tag } = view
  const visible = useMemo(() => applyView(nodes, { sort, status, tag }), [nodes, sort, status, tag])
  const visibleGroups = useMemo(
    () => groups.map((g) => ({ ...g, children: applyView(g.children, { sort, status, tag }) })),
    [groups, sort, status, tag],
  )
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
              visibleGroups
                .filter((g) => g.children.length > 0)
                .map((g) => (
                  <div key={g.dir} data-testid="content-root">
                    {labelled && (
                      <div className="flex items-center gap-1.5 px-2 pt-2 pb-0.5 text-[0.6875rem] font-medium tracking-wide text-muted-foreground uppercase">
                        <span className="truncate">{g.dir}</span>
                        {g.lang && (
                          <span className="rounded bg-muted px-1 normal-case">{g.lang}</span>
                        )}
                      </div>
                    )}
                    <TreeRoot label={labelled ? `Pages in ${g.dir}` : 'Pages'}>
                      {g.children.map((child) => (
                        <ContentTree
                          key={child.path}
                          node={child}
                          selected={selected}
                          onSelect={(path) => void load(path)}
                          onRename={openRename}
                          onDelete={openDelete}
                          onNewPage={(node) =>
                            openNewPage(
                              folderOf(node.isDir ? `${node.path}/x` : node.path, g.dir || contentDir),
                            )
                          }
                          errorPaths={errorPaths}
                        />
                      ))}
                    </TreeRoot>
                  </div>
                ))
            ) : tree === null ? (
              <SkeletonRows count={7} label="Loading pages" />
            ) : filtered ? (
              <EmptyState
                icon={ListFilter}
                text="No pages match this filter."
                action={{ label: 'Show all pages', onClick: view.reset }}
              />
            ) : (
              <EmptyState
                icon={FilePlus}
                text="No content files yet."
                action={{ label: 'New page', onClick: () => openNewPage('') }}
              />
            )}
          </nav>
        </ScrollArea>
      </div>
    </TooltipProvider>
  )
}
