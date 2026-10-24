import { useEffect, useRef } from 'react'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { markdown } from '@codemirror/lang-markdown'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap } from '@codemirror/view'

interface Props {
  value: string
  onChange: (text: string) => void
  onSave: () => void
}

/** CodeMirror 6 raw markdown editor. Remount (via key) to load a different document. */
export function MarkdownEditor({ value, onChange, onSave }: Props) {
  const host = useRef<HTMLDivElement>(null)
  const view = useRef<EditorView | null>(null)
  // Latest callbacks, so the editor is created once.
  const initial = useRef(value)
  // CodeMirror normalises line breaks to LF unless told the separator.
  const sep = useRef(value.includes('\r\n') ? '\r\n' : '\n')
  const handlers = useRef({ onChange, onSave })
  useEffect(() => {
    handlers.current = { onChange, onSave }
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
            ...defaultKeymap,
            ...historyKeymap,
          ]),
          EditorState.lineSeparator.of(sep.current),
          history(),
          markdown(),
          EditorView.lineWrapping,
          EditorView.updateListener.of((u) => {
            if (u.docChanged) {
              handlers.current.onChange(u.state.sliceDoc())
            }
          }),
        ],
      }),
    })
    view.current = v
    return () => {
      v.destroy()
      view.current = null
    }
    // Created once per mount; value is only the initial document.
     
  }, [])

  // Adopt external changes to value (e.g. after reload) without echoing them.
  useEffect(() => {
    const v = view.current
    if (v && v.state.sliceDoc() !== value) {
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } })
    }
  }, [value])

  return <div ref={host} className="rounded-md border font-mono text-sm" />
}
