import { create } from 'zustand'
import { api, setProjectID } from '@/api/client'
import { asApiError, type ApiError, type ContentNode, type Project } from '@/api/types'

interface ProjectState {
  project: Project | null
  tree: ContentNode | null
  loading: boolean
  error: ApiError | null

  open: (path: string) => Promise<boolean>
  restore: () => Promise<void>
  close: () => Promise<void>
  loadTree: () => Promise<void>
  reset: () => void
}

export const useProjectStore = create<ProjectState>()((set, get) => ({
  project: null,
  tree: null,
  loading: false,
  error: null,

  // Opens a project folder and loads its content tree.
  async open(path) {
    set({ loading: true, error: null })
    try {
      const opened = await api.openProject(path)
      set({ project: opened })
      setProjectID(opened.id)
      await get().loadTree()
      return true
    } catch (err) {
      get().reset()
      set({ error: asApiError(err) })
      return false
    } finally {
      set({ loading: false })
    }
  },

  // Restores the project the backend already has open, if any.
  async restore() {
    set({ loading: true })
    try {
      const current = await api.currentProject()
      set({ project: current })
      setProjectID(current.id)
      await get().loadTree()
    } catch (err) {
      // 'no_project' is the normal state on a fresh start, not a failure
      const apiErr = asApiError(err)
      get().reset()
      if (apiErr.code !== 'no_project') {
        set({ error: apiErr })
      }
    } finally {
      set({ loading: false })
    }
  },

  async close() {
    try {
      await api.closeProject()
    } finally {
      get().reset()
    }
  },

  // Reloads the content tree of the open project
  async loadTree() {
    if (get().project === null) {
      return
    }
    const { root } = await api.contentTree()
    set({ tree: root })
  },

  reset() {
    set({ project: null, tree: null, error: null })
    setProjectID(null)
  },
}))

export const selectIsOpen = (state: ProjectState): boolean => state.project !== null
