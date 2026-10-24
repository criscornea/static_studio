import { useEffect, useMemo, useRef, useState } from 'react'
import { Plus, Search, X } from 'lucide-react'

import { ContentTree } from '@/components/ContentTree'
import { DeletePageDialog } from '@/components/DeletePageDialog'
import { NewPageDialog } from '@/components/NewPageDialog'
import { RenamePageDialog } from '@/components/RenamePageDialog'
import { SearchResults } from '@/components/SearchResults'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import { applyView, collectTags, isDefaultView, type SortKey, type StatusFilter } from '@/lib/tree'
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

// Radix Select cannot use an empty string as a value.
const allTags = '__all__'

export function Sidebar() {
  const tree = useProjectStore((s) => s.tree)
  const selected = usePageStore((s) => s.path)
  const problems = useBuildStore((s) => s.problems)
  const errorPaths = useMemo(() => new Set(problems.map((p) => p.file)), [problems])
  const load = usePageStore((s) => s.load)

  const search = useSearchStore()
  const view = useViewStore()

  const searchBox = useRef<HTMLInputElement>(null)
  const [creating, setCreating] = useState(false)
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

  const nodes = useMemo(() => tree?.children ?? [], [tree])
  const tags = useMemo(() => collectTags(nodes), [nodes])
  const { sort, status, tag } = view
  const visible = useMemo(() => applyView(nodes, { sort, status, tag }), [nodes, sort, status, tag])
  const filtered = !isDefaultView({ sort, status, tag })

  return (
    <div className="flex h-full flex-col">
      <div className="space-y-2 border-b p-2">
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={searchBox}
              type="search"
              value={search.query}
              onChange={(e) => search.setQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Escape' && search.clear()}
              placeholder="Search pages  (Ctrl+K)"
              aria-label="Search pages"
              spellCheck={false}
              autoComplete="off"
              className="h-8 px-7 [&::-webkit-search-cancel-button]:hidden"
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
          <Button
            variant="outline"
            size="icon-sm"
            aria-label="New page"
            title="New page"
            onClick={() => setCreating(true)}
          >
            <Plus />
          </Button>
        </div>

        {!searching && (
          <div className="grid grid-cols-2 gap-2">
            <Select value={sort} onValueChange={(v) => view.setSort(v as SortKey)}>
              <SelectTrigger size="sm" className="w-full" aria-label="Sort by">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(sortLabels).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={status} onValueChange={(v) => view.setStatus(v as StatusFilter)}>
              <SelectTrigger size="sm" className="w-full" aria-label="Show">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(statusLabels).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {tags.length > 0 && (
              <Select
                value={tag ?? allTags}
                onValueChange={(v) => view.setTag(v === allTags ? null : v)}
              >
                <SelectTrigger size="sm" className="col-span-2 w-full" aria-label="Tag">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={allTags}>All tags</SelectItem>
                  {tags.map((t) => (
                    <SelectItem key={t} value={t}>
                      {t}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
        )}
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <nav className="p-2">
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
            <ul>
              {visible.map((child) => (
                <ContentTree
                  key={child.path}
                  node={child}
                  selected={selected}
                  onSelect={(path) => void load(path)}
                  onRename={setRenaming}
                  onDelete={setDeleting}
                  errorPaths={errorPaths}
                />
              ))}
            </ul>
          ) : (
            <div className="space-y-2 text-sm text-muted-foreground">
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

      <NewPageDialog open={creating} onOpenChange={setCreating} />
      <RenamePageDialog path={renaming} onClose={() => setRenaming(null)} />
      <DeletePageDialog path={deleting} onClose={() => setDeleting(null)} />
    </div>
  )
}
