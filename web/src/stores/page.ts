import { create } from 'zustand'
import { api } from '@/api/client'
import { asApiError, type ApiError, type Page, type PageFields } from '@/api/types'
import { useProjectStore } from './project'

interface PageState {
  // The path most recently requested, set before the response arrives.
  path: string | null
  page: Page | null
  loading: boolean
  error: ApiError | null
  // Editor text; null until a page is loaded. Differs from page.body when unsaved.
  draft: string | null
  // Form values; null until a page is loaded. Differs from page.fields when unsaved.
  fieldsDraft: PageFields | null
  // Path being saved, or null. Only a save of the same path is blocked.
  savingPath: string | null
  // Why the last save failed, including a hash conflict (code page_conflict).
  saveError: ApiError | null

  setDraft: (text: string) => void
  setFieldsDraft: (fields: PageFields) => void
  save: () => Promise<void>
  load: (path: string) => Promise<void>
  clear: () => void
}

export function fieldsEqual(a: PageFields, b: PageFields): boolean {
  return (
    a.title === b.title &&
    a.draft === b.draft &&
    (a.date ?? '') === (b.date ?? '') &&
    (a.tags ?? []).join('\u0000') === (b.tags ?? []).join('\u0000')
  )
}

// Incremented on every load. A response is only applied if no newer
// load has started since, so fast clicking can't show a stale page.
let latest = 0

export const usePageStore = create<PageState>()((set, get) => ({
  path: null,
  page: null,
  loading: false,
  error: null,
  draft: null,
  fieldsDraft: null,
  savingPath: null,
  saveError: null,

  async load(next) {
    const request = ++latest
    set({ path: next, loading: true, error: null, saveError: null })

    try {
      const loaded = await api.page(next)
      if (request !== latest) {
        return
      }
      set({ page: loaded, draft: loaded.body, fieldsDraft: loaded.fields })
    } catch (err) {
      if (request !== latest) {
        return
      }
      set({ page: null, draft: null, fieldsDraft: null, error: asApiError(err) })
    } finally {
      if (request === latest) {
        set({ loading: false })
      }
    }
  },

  setDraft(text) {
    set({ draft: text })
  },

  setFieldsDraft(fields) {
    set({ fieldsDraft: fields })
  },

  async save() {
    const { page, draft, fieldsDraft, savingPath } = get()
    if (page === null || draft === null || savingPath === page.path) {
      return
    }
    set({ savingPath: page.path, saveError: null })
    // Only send fields the user changed, so a plain body save never rewrites frontmatter.
    const fields =
      fieldsDraft !== null && !fieldsEqual(fieldsDraft, page.fields) ? fieldsDraft : undefined
    try {
      const { hash } = await api.savePage(page.path, draft, page.hash, fields)
      const current = get().page
      const saved = fields ?? current?.fields
      if (current === page) {
        set({ page: { ...page, body: draft, hash, fields: fields ?? page.fields } })
      } else if (current?.path === page.path && saved) {
        // Page was reloaded meanwhile: merge only what the save changed.
        set({ page: { ...current, body: draft, hash, fields: saved } })
      }
    } catch (err) {
      if (get().page?.path === page.path) {
        set({ saveError: asApiError(err) })
      }
    } finally {
      if (get().savingPath === page.path) {
        set({ savingPath: null })
      }
    }
  },

  clear() {
    latest++
    set({ path: null, page: null, draft: null, fieldsDraft: null, error: null, saveError: null, loading: false })
  },
}))

// A different project, or none, means the open page no longer applies.
useProjectStore.subscribe((state, prev) => {
  if (state.project?.id !== prev.project?.id) {
    usePageStore.getState().clear()
  }
})
