import { create } from 'zustand'

const key = 'studio.layout'

export const defaultSidebarSize = 22

interface Saved {
  sidebarSize: number
  sidebarCollapsed: boolean
}

interface LayoutState extends Saved {
  toggleSidebar: () => void
  /** Sidebar width as a percentage of the work area. */
  setSidebarSize: (percent: number) => void
}

function read(): Saved {
  const fallback = { sidebarSize: defaultSidebarSize, sidebarCollapsed: false }
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) {
      return fallback
    }
    const v = JSON.parse(raw) as Partial<Saved>
    return {
      sidebarSize:
        typeof v.sidebarSize === 'number' && v.sidebarSize >= 5 && v.sidebarSize <= 90
          ? v.sidebarSize
          : fallback.sidebarSize,
      sidebarCollapsed: v.sidebarCollapsed === true,
    }
  } catch {
    return fallback
  }
}

/** Shell layout that survives a reload. */
export const useLayoutStore = create<LayoutState>()((set) => ({
  ...read(),
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setSidebarSize: (percent) => set({ sidebarSize: percent }),
}))

useLayoutStore.subscribe(({ sidebarSize, sidebarCollapsed }) => {
  try {
    localStorage.setItem(key, JSON.stringify({ sidebarSize, sidebarCollapsed }))
  } catch {
    // Storage may be blocked; the layout then lasts for this session only.
  }
})
