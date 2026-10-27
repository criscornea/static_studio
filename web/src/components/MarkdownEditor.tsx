import { useEffect, useRef } from 'react'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { Compartment, EditorState } from '@codemirror/state'
import { EditorView, keymap } from '@codemirror/view'

import { FormatToolbar } from '@/components/FormatToolbar'
import { insertLink, toggleBold, toggleItalic } from '@/lib/editorFormat'
import { editorTheme } from '@/lib/editorTheme'
import { activeFormats } from '@/lib/formatState'
import { imageMarkdown } from '@/lib/markdown'
import { useEditorStore } from '@/stores/editor'
import { useThemeStore } from '@/stores/theme'

interface Props {
  value: string
  onChange: (text: string) => void
  onSave: () => void
  /** Uploads an image and resolves to its site path. Enables image drop. */
  onUploadImage?: (file: File) => Promise<string>
  onUploadError?: (err: unknown) => void
  /** Moves the cursor to this 1-based line when jumpLine or jumpToken changes. */
  jumpLine?: number | null
  /** A new value re-triggers the jump to the same line. */
  jumpToken?: unknown
}

/** CodeMirror 6 raw markdown editor. Remount (via key) to load a different document. */
export function MarkdownEditor({
  value,
  onChange,
  onSave,
  onUploadImage,
  onUploadError,
  jumpLine,
  jumpToken,
}: Props) {
  const host = useRef<HTMLDivElement>(null)
  const view = useRef<EditorView | null>(null)
  // Latest callbacks, so the editor is created once.
  const initial = useRef(value)
  // CodeMirror normalises line breaks to LF unless told the separator.
  const sep = useRef(value.includes('\r\n') ? '\r\n' : '\n')
  // Insertion points of in-flight drops, mapped through every document change.
  const trackers = useRef(new Set<{ pos: number }>())
  const theme = useRef(new Compartment())
  const dark = useThemeStore((s) => s.resolved === 'dark')
  const darkNow = useRef(dark)
  const handlers = useRef({ onChange, onSave, onUploadImage, onUploadError })
  useEffect(() => {
    handlers.current = { onChange, onSave, onUploadImage, onUploadError }
  })

  useEffect(() => {
    const v = new EditorView({
      parent: host.current!,
      state: EditorState.create({
        doc: initial.current,
        extensions: [
          keymap.of([
            {
              key: 'Mod-s',
              preventDefault: true,
              run: () => {
                handlers.current.onSave()
                return true
              },
            },
            // Ahead of the defaults (Mod-i) and of the global handler, which skips
            // keys the editor already handled (Mod-b toggles the sidebar elsewhere).
            { key: 'Mod-b', preventDefault: true, run: toggleBold },
            { key: 'Mod-i', preventDefault: true, run: toggleItalic },
            { key: 'Mod-k', preventDefault: true, run: insertLink },
            ...defaultKeymap,
            ...historyKeymap,
          ]),
          EditorState.lineSeparator.of(sep.current),
          theme.current.of(editorTheme(darkNow.current)),
          history(),
          markdown({ base: markdownLanguage }),
          EditorView.lineWrapping,
          EditorView.domEventHandlers({
            dragover: (e) => {
              if (handlers.current.onUploadImage && hasImage(e.dataTransfer)) {
                e.preventDefault()
                return true
              }
              return false
            },
            drop: (e, v) => {
              const upload = handlers.current.onUploadImage
              const files = Array.from(e.dataTransfer?.files ?? []).filter((f) =>
                f.type.startsWith('image/'),
              )
              if (!upload || files.length === 0) return false
              e.preventDefault()
              const pos = dropPosition(v, e)
              void dropImages(
                v,
                trackers.current,
                pos,
                files,
                upload,
                handlers.current.onUploadError,
              )
              return true
            },
          }),
          EditorView.domEventHandlers({
            focus: () => {
              const store = useEditorStore.getState()
              store.attach(view.current!)
              store.setFocused(true)
              store.setActive([...activeFormats(view.current!.state)])
            },
            blur: (e) => {
              const next = e.relatedTarget
              if (next instanceof Element && next.closest('[data-format-toolbar]')) return
              useEditorStore.getState().setFocused(false)
            },
          }),
          EditorView.updateListener.of((u) => {
            if (u.selectionSet || u.docChanged) {
              if (u.view.hasFocus) useEditorStore.getState().setActive([...activeFormats(u.state)])
            }
            if (u.docChanged) {
              for (const t of trackers.current) t.pos = u.changes.mapPos(t.pos, 1)
              handlers.current.onChange(u.state.sliceDoc())
            }
          }),
        ],
      }),
    })
    view.current = v
    return () => {
      useEditorStore.getState().detach(v)
      v.destroy()
      view.current = null
    }
    // Created once per mount; value is only the initial document.
  }, [])

  useEffect(() => {
    const v = view.current
    if (v && jumpLine) {
      const line = v.state.doc.line(Math.min(Math.max(jumpLine, 1), v.state.doc.lines))
      v.dispatch({
        selection: { anchor: line.from },
        effects: EditorView.scrollIntoView(line.from, { y: 'center' }),
      })
      v.focus()
    }
  }, [jumpLine, jumpToken])

  useEffect(() => {
    darkNow.current = dark
    view.current?.dispatch({ effects: theme.current.reconfigure(editorTheme(dark)) })
  }, [dark])

  // Adopt external changes to value (e.g. after reload) without echoing them.
  useEffect(() => {
    const v = view.current
    if (v && v.state.sliceDoc() !== value) {
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } })
    }
  }, [value])

  return (
    <div className="relative h-full min-h-0">
      <div ref={host} className="h-full min-h-0 overflow-hidden" />
      <FormatToolbar />
    </div>
  )
}

function dropPosition(v: EditorView, e: DragEvent): number {
  try {
    return v.posAtCoords({ x: e.clientX, y: e.clientY }) ?? v.state.selection.main.head
  } catch {
    // Layout is unavailable (e.g. a detached view): fall back to the cursor.
    return v.state.selection.main.head
  }
}

function hasImage(dt: DataTransfer | null): boolean {
  return Array.from(dt?.items ?? []).some((i) => i.kind === 'file' && i.type.startsWith('image/'))
}

async function dropImages(
  v: EditorView,
  trackers: Set<{ pos: number }>,
  pos: number,
  files: File[],
  upload: (file: File) => Promise<string>,
  onError?: (err: unknown) => void,
) {
  // The insertion point follows edits made while the uploads run.
  const tracker = { pos }
  trackers.add(tracker)
  try {
    for (const file of files) {
      try {
        const path = await upload(file)
        if (!v.dom.isConnected) {
          onError?.(
            new Error(`Image saved as ${path} but the editor was closed before it was linked.`),
          )
          continue
        }
        const at = Math.min(tracker.pos, v.state.doc.length)
        const lead = v.state.doc.lineAt(at).from === at ? '' : '\n'
        const text = lead + imageMarkdown(path, file.name) + '\n'
        v.dispatch({ changes: { from: at, insert: text }, selection: { anchor: at + text.length } })
      } catch (err) {
        onError?.(err)
      }
    }
  } finally {
    trackers.delete(tracker)
  }
}
