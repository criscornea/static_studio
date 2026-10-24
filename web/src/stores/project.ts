import { create } from 'zustand'
import { api, setProjectID } from '@/api/client'
import {
  asApiError,
  type ApiError,
  type ContentNode,
  type ContentRoot,
  type PageFields,
  type Project,
} from '@/api/types'
import { useBuildStore } from '@/stores/build'
import { useDevServerStore } from '@/stores/devserver'
import { usePublishStore } from '@/stores/publish'

interface ProjectState {
  project: Project | null
  // The primary content tree.
  tree: ContentNode | null
  // One tree per content dir, primary first.
  roots: ContentRoot[]
  loading: boolean
  error: ApiError | null

  open: (path: string) => Promise<boolean>
  restore: () => Promise<void>
  close: () => Promise<void>
  loadTree: () => Promise<void>
  // Updates one file's frontmatter in the loaded trees, without a reload.
  patchNodeFields: (path: string, fields: PageFields) => void
  reset: () => void
}

// Registered by the page store: resolves false when the user keeps unsaved edits.
let beforeLeave: (() => Promise<boolean>) | null = null
export function setBeforeLeave(fn: (() => Promise<boolean>) | null) {
  beforeLeave = fn
}

export const useProjectStore = create<ProjectState>()((set, get) => ({
  project: null,
  tree: null,
  roots: [],
  loading: false,
  error: null,

  // Opens a project folder and loads its content tree.
  async open(path) {
    if (get().project !== null && beforeLeave !== null && !(await beforeLeave())) {
      return false
    }
    set({ loading: true, error: null })
    try {
      const opened = await api.openProject(path)
      useBuildStore.getState().reset()
      useDevServerStore.getState().reset()
    usePublishStore.getState().reset()
      usePublishStore.getState().reset()
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
    if (beforeLeave !== null && !(await beforeLeave())) {
      return
    }
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
    const { root, roots } = await api.contentTree()
    set({ tree: root, roots: roots ?? [] })
  },

  patchNodeFields(path, fields) {
    const patch = (node: ContentNode): ContentNode => {
      if (!node.isDir) {
        return node.path === path ? { ...node, fields } : node
      }
      if (!node.children) {
        return node
      }
      const children = node.children.map(patch)
      return children.some((c, i) => c !== node.children![i]) ? { ...node, children } : node
    }
    const { tree, roots } = get()
    set({
      tree: tree && patch(tree),
      roots: roots.map((r) => ({ ...r, root: patch(r.root) })),
    })
  },

  reset() {
    useBuildStore.getState().reset()
    useDevServerStore.getState().reset()
    usePublishStore.getState().reset()
    set({ project: null, tree: null, roots: [], error: null })
    setProjectID(null)
  },
}))

export const selectIsOpen = (state: ProjectState): boolean => state.project !== null
