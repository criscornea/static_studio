import type { EditorView } from '@codemirror/view'
import { create } from 'zustand'

import { formatCommands } from '@/lib/editorFormat'
import type { FormatId } from '@/lib/formatState'

interface EditorState {
  /** The last focused editor; kept after blur so palette commands still reach it. */
  view: EditorView | null
  /** Whether the editor (or its toolbar) has focus. */
  focused: boolean
  /** Formats active at the selection of the last focused editor. */
  active: FormatId[]

  attach: (view: EditorView) => void
  detach: (view: EditorView) => void
  setFocused: (focused: boolean) => void
  setActive: (active: FormatId[]) => void
  /** Applies a format to the last focused editor and gives it focus back. */
  format: (id: FormatId) => void
}

export const useEditorStore = create<EditorState>()((set, get) => ({
  view: null,
  focused: false,
  active: [],

  attach: (view) => set({ view }),
  detach: (view) => {
    if (get().view === view) set({ view: null, focused: false, active: [] })
  },
  setFocused: (focused) => set({ focused }),
  setActive: (active) => {
    const cur = get().active
    if (cur.length === active.length && cur.every((a, i) => a === active[i])) return
    set({ active })
  },
  format: (id) => {
    const view = get().view
    if (!view?.dom.isConnected) return
    formatCommands[id](view)
    view.focus()
    // A closing dialog may hand focus elsewhere after this call.
    setTimeout(() => view.dom.isConnected && view.focus(), 0)
  },
}))
