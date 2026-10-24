import { create } from 'zustand'

import { defaultView, type TreeView } from '@/lib/tree'
import { useProjectStore } from './project'

interface ViewState extends TreeView {
  // Bumped to tell every folder in the tree to collapse.
  collapseEpoch: number
  collapseAll: () => void
  setSort: (sort: TreeView['sort']) => void
  setStatus: (status: TreeView['status']) => void
  setTag: (tag: string | null) => void
  reset: () => void
}

/** How the content tree is sorted and filtered. */
export const useViewStore = create<ViewState>()((set) => ({
  ...defaultView,
  collapseEpoch: 0,
  collapseAll: () => set((s) => ({ collapseEpoch: s.collapseEpoch + 1 })),
  setSort: (sort) => set({ sort }),
  setStatus: (status) => set({ status }),
  setTag: (tag) => set({ tag }),
  reset: () => set(defaultView),
}))

// A tag from another project means nothing here.
useProjectStore.subscribe((state, prev) => {
  if (state.project?.id !== prev.project?.id) {
    useViewStore.getState().reset()
  }
})
