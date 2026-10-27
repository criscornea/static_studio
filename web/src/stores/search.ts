import { create } from 'zustand'

import { api } from '@/api/client'
import { asApiError, type ApiError, type SearchResults } from '@/api/types'
import { useProjectStore } from './project'

/** How long typing must pause before the backend is asked. */
export const searchDelay = 250

interface SearchState {
  // What is in the search box, set at once on every keystroke.
  query: string
  // Results for the most recent query that was sent.
  results: SearchResults | null
  loading: boolean
  error: ApiError | null
  // Scroll offset of the results list, kept so it survives switching panels.
  scrollTop: number

  setScrollTop: (scrollTop: number) => void
  setQuery: (query: string) => void
  clear: () => void
}

// Incremented on every search and on clear. A response is only applied if
// nothing newer has started since, so slow replies can't overwrite fresh ones.
let latest = 0
let timer: ReturnType<typeof setTimeout> | undefined

export const useSearchStore = create<SearchState>()((set) => {
  async function run(query: string): Promise<void> {
    const request = ++latest
    set({ loading: true, error: null })

    try {
      const results = await api.search(query)
      if (request === latest) {
        set({ results })
      }
    } catch (err) {
      if (request === latest) {
        set({ results: null, error: asApiError(err) })
      }
    } finally {
      if (request === latest) {
        set({ loading: false })
      }
    }
  }

  return {
    query: '',
    results: null,
    loading: false,
    error: null,
    scrollTop: 0,

    setScrollTop(scrollTop) {
      set({ scrollTop })
    },

    setQuery(query) {
      clearTimeout(timer)
      set({ query, scrollTop: 0 })

      if (query.trim() === '') {
        latest++
        set({ results: null, error: null, loading: false })
        return
      }
      // Show the busy state straight away, not after the pause.
      set({ loading: true })
      timer = setTimeout(() => void run(query.trim()), searchDelay)
    },

    clear() {
      clearTimeout(timer)
      latest++
      set({ query: '', results: null, error: null, loading: false, scrollTop: 0 })
    },
  }
})

// Results from another project, or none, no longer apply.
useProjectStore.subscribe((state, prev) => {
  if (state.project?.id !== prev.project?.id) {
    useSearchStore.getState().clear()
  }
})
