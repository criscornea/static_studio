import type { ApiError, SearchResults as Results } from '@/api/types'
import { FileText } from 'lucide-react'

import { Highlight } from '@/components/Highlight'
import { cn } from '@/lib/utils'

interface SearchResultsProps {
  query: string
  results: Results | null
  loading: boolean
  error: ApiError | null
  selected: string | null
  onSelect: (path: string) => void
}

export function SearchResults({
  query,
  results,
  loading,
  error,
  selected,
  onSelect,
}: SearchResultsProps) {
  if (error) {
    return (
      <p role="alert" className="text-sm text-destructive">
        {error.message}
      </p>
    )
  }

  if (results === null) {
    return <p className="text-sm text-muted-foreground">Searching…</p>
  }

  if (results.hits.length === 0) {
    return (
      <p className={cn('text-sm text-muted-foreground', loading && 'opacity-50')}>
        No pages match “{query.trim()}”.
      </p>
    )
  }

  return (
    <div className={cn(loading && 'opacity-50')}>
      <ul>
        {results.hits.map((hit) => (
          <li key={hit.path}>
            <button
              type="button"
              title={hit.path}
              onClick={() => onSelect(hit.path)}
              className={cn(
                'block w-full px-2 py-1 text-left text-[0.8125rem] hover:bg-accent focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none focus-visible:ring-inset',
                selected === hit.path && 'bg-accent font-medium text-accent-foreground',
              )}
            >
              <span className="flex h-[1.4375rem] items-center gap-1 -my-0.5">
                <FileText className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="min-w-0 flex-1 truncate">
                  <Highlight
                    text={hit.title || hit.path.split('/').pop() || hit.path}
                    query={query}
                  />
                </span>
                {hit.draft && (
                  <span className="shrink-0" title="Draft">
                    <span aria-hidden className="block size-1.5 rounded-full bg-amber-500" />
                    <span className="sr-only">Draft</span>
                  </span>
                )}
              </span>
              <span className="block truncate pl-[1.1875rem] font-mono text-[0.6875rem] text-muted-foreground">
                {hit.path}
              </span>
              {hit.snippet && (
                <span className="line-clamp-1 block pl-[1.1875rem] text-xs text-muted-foreground">
                  <Highlight text={hit.snippet} query={query} />
                </span>
              )}
            </button>
          </li>
        ))}
      </ul>
      {results.truncated && (
        <p className="mt-2 px-3 text-xs text-muted-foreground">
          Showing the first {results.hits.length} matches. Refine your search to narrow them down.
        </p>
      )}
    </div>
  )
}
