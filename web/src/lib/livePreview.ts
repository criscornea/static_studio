import { ensureSyntaxTree, syntaxTree } from '@codemirror/language'
import { type EditorState, type Extension, type Range } from '@codemirror/state'
import {
  Decoration,
  EditorView,
  ViewPlugin,
  type DecorationSet,
  WidgetType,
  type ViewUpdate,
} from '@codemirror/view'
import type { SyntaxNode } from '@lezer/common'

export interface Span {
  from: number
  to: number
}

/** Kinds recorded in each decoration's spec so tests can tell them apart. */
export type LiveKind = 'hide' | 'mark' | 'line' | 'widget'

/** Replaces markup with a small piece of rendered DOM. Carries a label for tests. */
class TextWidget extends WidgetType {
  constructor(
    readonly label: string,
    readonly cls: string,
    readonly tag: 'span' | 'div' = 'span',
  ) {
    super()
  }
  eq(other: TextWidget) {
    return other.label === this.label && other.cls === this.cls && other.tag === this.tag
  }
  toDOM() {
    const el = document.createElement(this.tag)
    el.className = this.cls
    el.textContent = this.tag === 'span' ? this.label : ''
    return el
  }
}

function widget(w: TextWidget) {
  return Decoration.replace({ widget: w, kind: 'widget' satisfies LiveKind })
}

const hide = Decoration.replace({ kind: 'hide' satisfies LiveKind })

function mark(cls: string) {
  return Decoration.mark({ class: cls, kind: 'mark' satisfies LiveKind })
}

function line(cls: string) {
  return Decoration.line({ class: cls, kind: 'line' satisfies LiveKind })
}

const headingLevel = /^ATXHeading([1-6])$/

/** Line spans (start of first line to end of last) touched by any selection range. */
function touchedLines(state: EditorState): Span[] {
  return state.selection.ranges.map((r) => ({
    from: state.doc.lineAt(r.from).from,
    to: state.doc.lineAt(r.to).to,
  }))
}

/**
 * Decorations that render markdown inline. Markup is hidden except on lines
 * touched by a selection range. Only `visible` ranges are walked, and the
 * document is never changed.
 */
export function livePreviewDecorations(state: EditorState, visible?: Span[]): DecorationSet {
  const tree = ensureSyntaxTree(state, state.doc.length, 50) ?? syntaxTree(state)
  const touched = touchedLines(state)
  const ranges: Range<Decoration>[] = []
  const lined = new Set<string>()

  const onCursor = (from: number, to: number) => touched.some((t) => from <= t.to && to >= t.from)

  /** Hide [from, to) unless it sits on a cursor line. Empty spans are skipped. */
  const conceal = (from: number, to: number) => {
    if (to > from && !onCursor(from, to)) ranges.push(hide.range(from, to))
  }

  /** Hide a mark plus one following space, if any. */
  const concealWithSpace = (n: SyntaxNode) => {
    const next = state.sliceDoc(n.to, n.to + 1)
    conceal(n.from, next === ' ' ? n.to + 1 : n.to)
  }

  const addLine = (pos: number, cls: string) => {
    const l = state.doc.lineAt(pos)
    const key = `${l.from}:${cls}`
    if (lined.has(key)) return
    lined.add(key)
    ranges.push(line(cls).range(l.from))
  }

  const spans = visible ?? [{ from: 0, to: state.doc.length }]
  for (const span of spans) {
    tree.iterate({
      from: span.from,
      to: span.to,
      enter: (ref) => {
        const name = ref.name
        const node = ref.node
        const heading = headingLevel.exec(name)
        if (heading) {
          addLine(ref.from, `cm-lp-h${heading[1]}`)
        } else if (name === 'Emphasis') {
          ranges.push(mark('cm-lp-em').range(ref.from, ref.to))
        } else if (name === 'StrongEmphasis') {
          ranges.push(mark('cm-lp-strong').range(ref.from, ref.to))
        } else if (name === 'Strikethrough') {
          ranges.push(mark('cm-lp-strike').range(ref.from, ref.to))
        } else if (name === 'InlineCode') {
          ranges.push(mark('cm-lp-code').range(ref.from, ref.to))
        } else if (name === 'Link') {
          const marks = node.getChildren('LinkMark')
          // Inline links only: [text](url). Reference links stay as source.
          const [open, close, paren] = marks
          if (marks.length === 4 && open && close && paren && close.to === paren.from) {
            ranges.push(mark('cm-lp-link').range(open.to, close.from))
            conceal(open.from, open.to)
            conceal(close.from, ref.to)
          }
        } else if (name === 'Blockquote') {
          for (let p = ref.from; p <= ref.to; ) {
            const l = state.doc.lineAt(p)
            addLine(l.from, 'cm-lp-quote')
            p = l.to + 1
          }
        } else if (name === 'QuoteMark') {
          concealWithSpace(node)
        } else if (name === 'ListMark') {
          if (onCursor(ref.from, ref.to)) return
          const ordered = node.parent?.parent?.name === 'OrderedList'
          const text = state.sliceDoc(ref.from, ref.to)
          const w = ordered ? new TextWidget(text, 'cm-lp-num') : new TextWidget('\u2022', 'cm-lp-bullet')
          ranges.push(widget(w).range(ref.from, ref.to))
        } else if (name === 'HorizontalRule') {
          if (!onCursor(ref.from, ref.to)) {
            ranges.push(widget(new TextWidget('---', 'cm-lp-hr', 'div')).range(ref.from, ref.to))
          }
        } else if (name === 'HeaderMark' && node.parent && headingLevel.test(node.parent.name)) {
          concealWithSpace(node)
        } else if (name === 'EmphasisMark' || name === 'StrikethroughMark') {
          conceal(ref.from, ref.to)
        } else if (name === 'CodeMark' && node.parent?.name === 'InlineCode') {
          conceal(ref.from, ref.to)
        }
      },
    })
  }

  return Decoration.set(ranges, true)
}

const plugin = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    constructor(view: EditorView) {
      this.decorations = build(view)
    }
    update(u: ViewUpdate) {
      if (u.docChanged || u.selectionSet || u.viewportChanged || u.transactions.length > 0) {
        this.decorations = build(u.view)
      }
    }
  },
  { decorations: (p) => p.decorations },
)

function build(view: EditorView): DecorationSet {
  return livePreviewDecorations(
    view.state,
    view.visibleRanges.map((r) => ({ from: r.from, to: r.to })),
  )
}

const theme = EditorView.baseTheme({
  '.cm-lp-h1': { fontSize: '1.7em', fontWeight: '700', lineHeight: '1.4' },
  '.cm-lp-h2': { fontSize: '1.45em', fontWeight: '700', lineHeight: '1.4' },
  '.cm-lp-h3': { fontSize: '1.25em', fontWeight: '600', lineHeight: '1.4' },
  '.cm-lp-h4, .cm-lp-h5, .cm-lp-h6': { fontSize: '1.1em', fontWeight: '600' },
  '.cm-lp-strong': { fontWeight: '700' },
  '.cm-lp-em': { fontStyle: 'italic' },
  '.cm-lp-strike': { textDecoration: 'line-through' },
  '.cm-lp-link': {
    color: 'oklch(0.5 0.15 255)',
    textDecoration: 'underline',
  },
  '&dark .cm-lp-link': { color: 'oklch(0.75 0.12 250)' },
  '.cm-lp-quote': {
    borderLeft: '3px solid var(--border)',
    color: 'var(--muted-foreground)',
    paddingLeft: '0.5rem',
  },
  '.cm-lp-bullet, .cm-lp-num': { color: 'var(--muted-foreground)', display: 'inline-block' },
  '.cm-lp-bullet': { width: '1ch', textAlign: 'center' },
  '.cm-lp-hr': {
    display: 'block',
    height: '0',
    margin: '0.7em 0',
    borderTop: '2px solid var(--border)',
  },
  '.cm-lp-code': {
    backgroundColor: 'color-mix(in oklab, var(--muted) 70%, transparent)',
    borderRadius: '3px',
    padding: '0 0.2em',
  },
})

/** Render markdown inline with markup revealed only on the cursor lines. */
export function livePreview(): Extension {
  return [plugin, theme]
}
