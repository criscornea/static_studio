import { create } from 'zustand'
import { api } from '@/api/client'
import { asApiError, type ApiError, type Page } from '@/api/types'
import { useProjectStore } from './project'

interface PageState {
  // The path most recently requested, set before the response arrives.
  path: string | null
  page: Page | null
  loading: boolean
  error: ApiError | null

  load: (path: string) => Promise<void>
  clear: () => void
}

// Incremented on every load. A response is only applied if no newer
// load has started since, so fast clicking can't show a stale page.
let latest = 0

export const usePageStore = create<PageState>()((set) => ({
  path: null,
  page: null,
  loading: false,
  error: null,

  async load(next) {
    const request = ++latest
    set({ path: next, loading: true, error: null })

    try {
      const loaded = await api.page(next)
      if (request !== latest) {
        return
      }
      set({ page: loaded })
    } catch (err) {
      if (request !== latest) {
        return
      }
      set({ page: null, error: asApiError(err) })
    } finally {
      if (request === latest) {
        set({ loading: false })
      }
    }
  },

  clear() {
    latest++
    set({ path: null, page: null, error: null, loading: false })
  },
}))

// A different project, or none, means the open page no longer applies.
useProjectStore.subscribe((state, prev) => {
  if (state.project?.id !== prev.project?.id) {
    usePageStore.getState().clear()
  }
})
