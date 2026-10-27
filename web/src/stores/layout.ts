import { create } from 'zustand'

const key = 'studio.layout'

export const defaultSidebarSize = 22
export const defaultPreviewSize = 45
export const defaultBuildLogSize = 30

export type SidebarPanel = 'files' | 'search'

interface Saved {
  /** Which panel the sidebar shows. */
  sidebarPanel: SidebarPanel
  sidebarSize: number
  sidebarCollapsed: boolean
  /** Preview width as a percentage of the editor area. */
  previewSize: number
  /** Build log height as a percentage of the main area. */
  buildLogSize: number
}

interface LayoutState extends Saved {
  /** Whether the preview pane is shown. Not persisted: it needs a running server. */
  previewOpen: boolean
  /** Whether the build log panel is shown. */
  buildLogOpen: boolean
  toggleSidebar: () => void
  /** Shows a panel and expands the sidebar if it was collapsed. */
  showSidebarPanel: (panel: SidebarPanel) => void
  /** Sidebar width as a percentage of the work area. */
  setSidebarSize: (percent: number) => void
  togglePreview: () => void
  setPreviewOpen: (open: boolean) => void
  setPreviewSize: (percent: number) => void
  toggleBuildLog: () => void
  setBuildLogOpen: (open: boolean) => void
  setBuildLogSize: (percent: number) => void
}

function size(v: unknown, fallback: number): number {
  return typeof v === 'number' && v >= 5 && v <= 90 ? v : fallback
}

function read(): Saved {
  const fallback: Saved = {
    sidebarPanel: 'files',
    sidebarSize: defaultSidebarSize,
    sidebarCollapsed: false,
    previewSize: defaultPreviewSize,
    buildLogSize: defaultBuildLogSize,
  }
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) {
      return fallback
    }
    const v = JSON.parse(raw) as Partial<Saved>
    return {
      sidebarPanel: v.sidebarPanel === 'search' ? 'search' : 'files',
      sidebarSize: size(v.sidebarSize, fallback.sidebarSize),
      sidebarCollapsed: v.sidebarCollapsed === true,
      previewSize: size(v.previewSize, fallback.previewSize),
      buildLogSize: size(v.buildLogSize, fallback.buildLogSize),
    }
  } catch {
    return fallback
  }
}

/** Shell layout that survives a reload. */
export const useLayoutStore = create<LayoutState>()((set) => ({
  ...read(),
  previewOpen: false,
  buildLogOpen: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  showSidebarPanel: (panel) => set({ sidebarPanel: panel, sidebarCollapsed: false }),
  setSidebarSize: (percent) => set({ sidebarSize: percent }),
  togglePreview: () => set((s) => ({ previewOpen: !s.previewOpen })),
  setPreviewOpen: (open) => set({ previewOpen: open }),
  setPreviewSize: (percent) => set({ previewSize: percent }),
  toggleBuildLog: () => set((s) => ({ buildLogOpen: !s.buildLogOpen })),
  setBuildLogOpen: (open) => set({ buildLogOpen: open }),
  setBuildLogSize: (percent) => set({ buildLogSize: percent }),
}))

useLayoutStore.subscribe((s) => {
  const saved: Saved = {
    sidebarPanel: s.sidebarPanel,
    sidebarSize: s.sidebarSize,
    sidebarCollapsed: s.sidebarCollapsed,
    previewSize: s.previewSize,
    buildLogSize: s.buildLogSize,
  }
  try {
    localStorage.setItem(key, JSON.stringify(saved))
  } catch {
    // Storage may be blocked; the layout then lasts for this session only.
  }
})
