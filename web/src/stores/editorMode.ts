import { create } from 'zustand'

export type EditorMode = 'live' | 'source'

export const editorModeKey = 'studio.editorMode'

interface EditorModeState {
  mode: EditorMode
  setMode: (mode: EditorMode) => void
  toggle: () => void
}

function read(): EditorMode {
  try {
    return localStorage.getItem(editorModeKey) === 'source' ? 'source' : 'live'
  } catch {
    return 'live'
  }
}

/** Live preview (markup hidden off the cursor lines) or plain source; persisted. */
export const useEditorModeStore = create<EditorModeState>()((set, get) => ({
  mode: read(),
  setMode(mode) {
    try {
      localStorage.setItem(editorModeKey, mode)
    } catch {
      // Storage may be blocked; the choice then lasts for this session only.
    }
    set({ mode })
  },
  toggle: () => get().setMode(get().mode === 'live' ? 'source' : 'live'),
}))
