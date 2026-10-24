import { create } from 'zustand'

interface UiState {
  /** Folder to preselect (undefined: the open page's folder) while the new page dialog is open. */
  newPage: { folder?: string; dir?: string } | null
  renamePath: string | null
  deletePath: string | null
  publishOpen: boolean
  paletteOpen: boolean
  quickOpenOpen: boolean
  shortcutsOpen: boolean
  /** Set by the focus-search command; the sidebar clears it once the box has focus. */
  searchFocusRequested: boolean

  openNewPage: (folder?: string, dir?: string) => void
  closeNewPage: () => void
  openRename: (path: string) => void
  closeRename: () => void
  openDelete: (path: string) => void
  closeDelete: () => void
  setPublishOpen: (open: boolean) => void
  setPaletteOpen: (open: boolean) => void
  setQuickOpenOpen: (open: boolean) => void
  setShortcutsOpen: (open: boolean) => void
  requestSearchFocus: () => void
  clearSearchFocus: () => void
}

/** Which dialogs and overlays are open. Lives in a store so commands can open them. */
export const useUiStore = create<UiState>()((set) => ({
  newPage: null,
  renamePath: null,
  deletePath: null,
  publishOpen: false,
  paletteOpen: false,
  quickOpenOpen: false,
  shortcutsOpen: false,
  searchFocusRequested: false,

  openNewPage: (folder, dir) =>
    set({ newPage: { folder: folder === '' ? undefined : folder, dir } }),
  closeNewPage: () => set({ newPage: null }),
  openRename: (path) => set({ renamePath: path }),
  closeRename: () => set({ renamePath: null }),
  openDelete: (path) => set({ deletePath: path }),
  closeDelete: () => set({ deletePath: null }),
  setPublishOpen: (open) => set({ publishOpen: open }),
  setPaletteOpen: (open) => set({ paletteOpen: open, quickOpenOpen: false }),
  setQuickOpenOpen: (open) => set({ quickOpenOpen: open, paletteOpen: false }),
  setShortcutsOpen: (open) => set({ shortcutsOpen: open }),
  requestSearchFocus: () => set({ searchFocusRequested: true }),
  clearSearchFocus: () => set({ searchFocusRequested: false }),
}))
