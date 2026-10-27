import { useEffect, useLayoutEffect, useRef } from 'react'
import { Search, X } from 'lucide-react'

import { SearchResults } from '@/components/SearchResults'
import { Input } from '@/components/ui/input'
import { usePageStore } from '@/stores/page'
import { useSearchStore } from '@/stores/search'
import { useUiStore } from '@/stores/ui'

/** Sidebar panel with the content search box and its results. */
export function SearchPanel() {
  const query = useSearchStore((s) => s.query)
  const results = useSearchStore((s) => s.results)
  const loading = useSearchStore((s) => s.loading)
  const error = useSearchStore((s) => s.error)
  const setQuery = useSearchStore((s) => s.setQuery)
  const clear = useSearchStore((s) => s.clear)
  const setScrollTop = useSearchStore((s) => s.setScrollTop)
  const selected = usePageStore((s) => s.path)
  const load = usePageStore((s) => s.load)
  const focusRequested = useUiStore((s) => s.searchFocusRequested)
  const clearSearchFocus = useUiStore((s) => s.clearSearchFocus)

  const box = useRef<HTMLInputElement>(null)
  const list = useRef<HTMLDivElement>(null)
  const searching = query.trim() !== ''

  // The focus-search command lands here, also right after the panel mounted.
  useEffect(() => {
    if (focusRequested) {
      box.current?.focus()
      box.current?.select()
      clearSearchFocus()
    }
  }, [focusRequested, clearSearchFocus])

  // Restore the scroll position kept from the last time the panel was open.
  useLayoutEffect(() => {
    if (list.current) {
      list.current.scrollTop = useSearchStore.getState().scrollTop
    }
  }, [])

  return (
    <div className="flex h-full min-h-0 flex-col bg-sidebar text-sidebar-foreground">
      <div className="space-y-1.5 border-b p-2">
        <h2 className="px-1 text-[0.6875rem] font-semibold tracking-wide text-muted-foreground uppercase">
          Search
        </h2>
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={box}
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Escape' && clear()}
            placeholder="Search  (Ctrl+Shift+F)"
            aria-label="Search pages"
            spellCheck={false}
            autoComplete="off"
            className="h-7 px-7 text-[0.8125rem] [&::-webkit-search-cancel-button]:hidden"
          />
          {searching && (
            <button
              type="button"
              aria-label="Clear search"
              onClick={clear}
              className="absolute top-1/2 right-2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            >
              <X className="size-3.5" />
            </button>
          )}
        </div>
      </div>

      <div
        ref={list}
        data-testid="search-scroll"
        onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
        className="min-h-0 flex-1 overflow-y-auto py-1"
      >
        {searching ? (
          <SearchResults
            query={query}
            results={results}
            loading={loading}
            error={error}
            selected={selected}
            onSelect={(path) => void load(path)}
          />
        ) : (
          <p className="px-3 py-2 text-xs text-muted-foreground">
            Type to search titles and page content.
          </p>
        )}
      </div>
    </div>
  )
}
