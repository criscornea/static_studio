import type { ApiError, SearchResults as Results } from '@/api/types'
import { Badge } from '@/components/ui/badge'
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
              onClick={() => onSelect(hit.path)}
              className={cn(
                'block w-full rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent',
                selected === hit.path && 'bg-accent text-accent-foreground',
              )}
            >
              <span className="flex items-center gap-2">
                <span className="min-w-0 flex-1 truncate font-medium">
                  <Highlight
                    text={hit.title || hit.path.split('/').pop() || hit.path}
                    query={query}
                  />
                </span>
                {hit.draft && (
                  <Badge variant="secondary" className="shrink-0">
                    Draft
                  </Badge>
                )}
              </span>
              <span className="block truncate font-mono text-[0.6875rem] text-muted-foreground">
                {hit.path}
              </span>
              {hit.snippet && (
                <span className="mt-0.5 line-clamp-2 block text-xs text-muted-foreground">
                  <Highlight text={hit.snippet} query={query} />
                </span>
              )}
            </button>
          </li>
        ))}
      </ul>
      {results.truncated && (
        <p className="mt-2 px-2 text-xs text-muted-foreground">
          Showing the first {results.hits.length} matches. Refine your search to narrow them down.
        </p>
      )}
    </div>
  )
}
