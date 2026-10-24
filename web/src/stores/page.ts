import { create } from 'zustand'
import { api } from '@/api/client'
import { asApiError, type ApiError, type Page, type PageFields } from '@/api/types'
import { setBeforeLeave, useProjectStore } from './project'

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
  // True while the draft or form differs from the last saved page.
  dirty: boolean

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

export const AUTOSAVE_DELAY_MS = 1500

function isDirty(page: Page | null, draft: string | null, fields: PageFields | null): boolean {
  if (page === null) {
    return false
  }
  return (
    (draft !== null && draft !== page.body) || (fields !== null && !fieldsEqual(fields, page.fields))
  )
}

let timer: ReturnType<typeof setTimeout> | null = null
// Saves run one after another, so two never carry the same hash.
let queue: Promise<void> = Promise.resolve()

export function cancelAutosave() {
  if (timer !== null) {
    clearTimeout(timer)
    timer = null
  }
}

// A conflict stops autosave until the user acts; other errors retry on the next edit.
function scheduleAutosave() {
  cancelAutosave()
  const { dirty, saveError } = usePageStore.getState()
  if (!dirty || saveError?.code === 'page_conflict') {
    return
  }
  timer = setTimeout(() => {
    timer = null
    void usePageStore.getState().save()
  }, AUTOSAVE_DELAY_MS)
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
  dirty: false,

  async load(next) {
    cancelAutosave()
    if (get().dirty) {
      await get().save()
      if (get().dirty && typeof window !== 'undefined' && typeof window.confirm === 'function') {
        if (!window.confirm('This page has unsaved changes. Discard them?')) {
          return
        }
      }
    }
    const request = ++latest
    // Drop the old page so it can't be edited while the new one loads.
    set({
      path: next,
      page: null,
      draft: null,
      fieldsDraft: null,
      dirty: false,
      loading: true,
      error: null,
      saveError: null,
    })

    try {
      const loaded = await api.page(next)
      if (request !== latest) {
        return
      }
      set({ page: loaded, draft: loaded.body, fieldsDraft: loaded.fields, dirty: false })
    } catch (err) {
      if (request !== latest) {
        return
      }
      set({ page: null, draft: null, fieldsDraft: null, dirty: false, error: asApiError(err) })
    } finally {
      if (request === latest) {
        set({ loading: false })
      }
    }
  },

  setDraft(text) {
    const { page, fieldsDraft } = get()
    set({ draft: text, dirty: isDirty(page, text, fieldsDraft) })
    scheduleAutosave()
  },

  setFieldsDraft(fields) {
    const { page, draft } = get()
    set({ fieldsDraft: fields, dirty: isDirty(page, draft, fields) })
    scheduleAutosave()
  },

  save() {
    cancelAutosave()
    const run = queue.then(() => doSave(get, set))
    queue = run.catch(() => undefined)
    return run
  },

  clear() {
    latest++
    cancelAutosave()
    set({
      path: null,
      page: null,
      draft: null,
      fieldsDraft: null,
      error: null,
      saveError: null,
      loading: false,
      dirty: false,
    })
  },
}))

type Get = () => PageState
type Set = (partial: Partial<PageState>) => void

async function doSave(get: Get, set: Set): Promise<void> {
  {
    const { page, draft, fieldsDraft } = get()
    if (page === null || draft === null || !get().dirty) {
      return
    }
    set({ savingPath: page.path, saveError: null })
    // Only send fields the user changed, so a plain body save never rewrites frontmatter.
    const fields =
      fieldsDraft !== null && !fieldsEqual(fieldsDraft, page.fields) ? fieldsDraft : undefined
    let ok = false
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
      ok = true
    } catch (err) {
      if (get().page?.path === page.path) {
        set({ saveError: asApiError(err) })
      }
    } finally {
      if (get().savingPath === page.path) {
        set({ savingPath: null })
      }
      const now = get()
      set({ dirty: isDirty(now.page, now.draft, now.fieldsDraft) })
      // Edits made while saving need another pass. After a failure, the next edit retries.
      if (ok) {
        scheduleAutosave()
      }
    }
  }
}

// A different project, or none, means the open page no longer applies.
useProjectStore.subscribe((state, prev) => {
  if (state.project?.id !== prev.project?.id) {
    usePageStore.getState().clear()
  }
})

// Save before leaving the project; ask before dropping edits that could not be saved.
setBeforeLeave(async () => {
  cancelAutosave()
  await usePageStore.getState().save()
  if (!usePageStore.getState().dirty) {
    return true
  }
  return typeof window !== 'undefined' && typeof window.confirm === 'function'
    ? window.confirm('This page has unsaved changes. Discard them?')
    : true
})
