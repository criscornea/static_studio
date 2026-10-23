import { create } from 'zustand'

import { defaultView, type TreeView } from '@/lib/tree'
import { useProjectStore } from './project'

interface ViewState extends TreeView {
  setSort: (sort: TreeView['sort']) => void
  setStatus: (status: TreeView['status']) => void
  setTag: (tag: string | null) => void
  reset: () => void
}

/** How the content tree is sorted and filtered. */
export const useViewStore = create<ViewState>()((set) => ({
  ...defaultView,
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
