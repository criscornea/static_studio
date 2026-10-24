import { useEffect, useMemo, useState } from 'react'
import { FileText } from 'lucide-react'

import { api } from '@/api/client'
import type { SearchHit } from '@/api/types'
import {
  CommandDialog,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
  Command as CommandRoot,
} from '@/components/ui/command'
import { flattenPages } from '@/lib/tree'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import { searchDelay } from '@/stores/search'
import { useUiStore } from '@/stores/ui'

const maxListed = 50

/** Ctrl/Cmd+P: jump to a page by title or path. */
export function QuickOpen() {
  const open = useUiStore((s) => s.quickOpenOpen)
  const setOpen = useUiStore((s) => s.setQuickOpenOpen)

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Open page"
      description="Search pages by title or path"
    >
      {open && <QuickOpenBody onDone={() => setOpen(false)} />}
    </CommandDialog>
  )
}

function QuickOpenBody({ onDone }: { onDone: () => void }) {
  const tree = useProjectStore((s) => s.tree)
  const load = usePageStore((s) => s.load)
  const [query, setQuery] = useState('')
  // Keyed by query so a reply always belongs to what is typed now.
  const [found, setFound] = useState<{ query: string; hits: SearchHit[] } | null>(null)
  const all = useMemo(() => flattenPages(tree?.children ?? []).slice(0, maxListed), [tree])
  const q = query.trim()

  useEffect(() => {
    if (q === '') {
      return
    }
    let cancelled = false
    const timer = setTimeout(() => {
      api
        .search(q)
        .then((r) => !cancelled && setFound({ query: q, hits: r.hits }))
        .catch(() => !cancelled && setFound({ query: q, hits: [] }))
    }, searchDelay)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [q])

  const pending = q !== '' && found?.query !== q
  const entries =
    q === ''
      ? all
      : (found?.hits ?? []).map((h) => ({
          path: h.path,
          title: h.title || h.path.split('/').pop() || h.path,
          draft: h.draft,
        }))

  return (
    <CommandRoot shouldFilter={false} label="Pages">
      <CommandInput value={query} onValueChange={setQuery} placeholder="Open page..." />
      <CommandList>
        {!pending && <CommandEmpty>No pages found.</CommandEmpty>}
        {entries.map((e) => (
          <CommandItem
            key={e.path}
            value={e.path}
            onSelect={() => {
              onDone()
              void load(e.path)
            }}
          >
            <FileText className="text-muted-foreground" />
            <span className="truncate">{e.title}</span>
            {e.draft && <span className="text-xs text-amber-600">Draft</span>}
            <span className="ml-auto truncate font-mono text-xs text-muted-foreground">
              {e.path}
            </span>
          </CommandItem>
        ))}
      </CommandList>
    </CommandRoot>
  )
}
