import { EditorSelection, type ChangeSpec, type EditorState } from '@codemirror/state'
import type { EditorView } from '@codemirror/view'

import { enclosing, type FormatId } from '@/lib/formatState'

/** A CodeMirror command: returns true when it handled the key or click. */
export type FormatCommand = (view: EditorView) => boolean

export const urlPlaceholder = 'url'

interface Inline {
  marker: string
  node: string
  mark: string
}

const inlines = {
  bold: { marker: '**', node: 'StrongEmphasis', mark: 'EmphasisMark' },
  italic: { marker: '*', node: 'Emphasis', mark: 'EmphasisMark' },
  strike: { marker: '~~', node: 'Strikethrough', mark: 'StrikethroughMark' },
  code: { marker: '`', node: 'InlineCode', mark: 'CodeMark' },
} satisfies Record<string, Inline>

/** Wraps the selection in the marker, or removes the markers when it is already wrapped. */
function toggleInline(spec: Inline): FormatCommand {
  return (view) => {
    const { state } = view
    view.dispatch(
      state.update(
        state.changeByRange((range) => {
          const node = enclosing(state, range.from, range.to, [spec.node])
          if (node) {
            const marks: { from: number; to: number }[] = []
            for (let c = node.firstChild; c; c = c.nextSibling) {
              if (c.name === spec.mark) marks.push({ from: c.from, to: c.to })
            }
            const shift = (pos: number) =>
              marks.reduce((p, m) => p - Math.max(0, Math.min(pos, m.to) - m.from), pos)
            return {
              changes: marks.map((m) => ({ from: m.from, to: m.to })),
              range: EditorSelection.range(shift(range.anchor), shift(range.head)),
            }
          }
          const n = spec.marker.length
          return {
            changes: [
              { from: range.from, insert: spec.marker },
              { from: range.to, insert: spec.marker },
            ],
            range: EditorSelection.range(range.anchor + n, range.head + n),
          }
        }),
        { scrollIntoView: true, userEvent: 'input.format' },
      ),
    )
    return true
  }
}

export const toggleBold = toggleInline(inlines.bold)
export const toggleItalic = toggleInline(inlines.italic)
export const toggleStrikethrough = toggleInline(inlines.strike)
export const toggleInlineCode = toggleInline(inlines.code)

/**
 * Inserts `[text](url)` around the selection with the url selected. Inside an
 * existing link it selects the url so it can be edited.
 */
export const insertLink: FormatCommand = (view) => {
  const { state } = view
  view.dispatch(
    state.update(
      state.changeByRange((range) => {
        const link = enclosing(state, range.from, range.to, ['Link'])
        if (link) {
          for (let c = link.firstChild; c; c = c.nextSibling) {
            if (c.name === 'URL') return { range: EditorSelection.range(c.from, c.to) }
          }
        }
        const text = state.sliceDoc(range.from, range.to)
        if (/^https?:\/\/\S+$/.test(text)) {
          // A pasted address becomes the target; the cursor lands in the label.
          return {
            changes: { from: range.from, to: range.to, insert: `[](${text})` },
            range: EditorSelection.cursor(range.from + 1),
          }
        }
        const label = text || 'link'
        const start = range.from + label.length + 3
        return {
          changes: { from: range.from, to: range.to, insert: `[${label}](${urlPlaceholder})` },
          range: EditorSelection.range(start, start + urlPlaceholder.length),
        }
      }),
      { scrollIntoView: true, userEvent: 'input.format' },
    ),
  )
  return true
}

interface LineInfo {
  from: number
  to: number
  text: string
}

/** Lines touched by the main selection; a selection ending at a line start excludes that line. */
function selectedLines(state: EditorState): LineInfo[] {
  const lines: LineInfo[] = []
  const seen = new Set<number>()
  for (const r of state.selection.ranges) {
    const first = state.doc.lineAt(r.from).number
    let last = state.doc.lineAt(r.to).number
    if (last > first && state.doc.line(last).from === r.to) last--
    for (let n = first; n <= last; n++) {
      if (seen.has(n)) continue
      seen.add(n)
      const l = state.doc.line(n)
      lines.push({ from: l.from, to: l.to, text: l.text })
    }
  }
  return lines
}

/** Blank lines are left alone unless every line is blank. */
function targets(lines: LineInfo[]): LineInfo[] {
  const filled = lines.filter((l) => l.text.trim() !== '')
  return filled.length > 0 ? filled : lines
}

function dispatchChanges(view: EditorView, changes: ChangeSpec[]): boolean {
  view.dispatch({ changes, scrollIntoView: true, userEvent: 'input.format' })
  return true
}

const headingRe = /^(#{1,6})[ \t]+/

function toggleHeading(level: number): FormatCommand {
  return (view) => {
    const lines = targets(selectedLines(view.state))
    const all = lines.every((l) => headingRe.exec(l.text)?.[1]?.length === level)
    const changes = lines.map((l) => {
      const m = headingRe.exec(l.text)
      const insert = all ? '' : '#'.repeat(level) + ' '
      return { from: l.from, to: l.from + (m ? m[0].length : 0), insert }
    })
    return dispatchChanges(view, changes)
  }
}

export const toggleHeading1 = toggleHeading(1)
export const toggleHeading2 = toggleHeading(2)
export const toggleHeading3 = toggleHeading(3)

const bulletRe = /^(\s*)[-*+][ \t]+/
const numberedRe = /^(\s*)\d+[.)][ \t]+/
const quoteRe = /^(\s*)>[ \t]?/

export const toggleBulletList: FormatCommand = (view) => {
  const lines = targets(selectedLines(view.state))
  const all = lines.every((l) => bulletRe.test(l.text))
  return dispatchChanges(
    view,
    lines.map((l) => replacePrefix(l, [bulletRe, numberedRe], all ? '' : '- ')),
  )
}

export const toggleNumberedList: FormatCommand = (view) => {
  const lines = targets(selectedLines(view.state))
  const all = lines.every((l) => numberedRe.test(l.text))
  return dispatchChanges(
    view,
    lines.map((l, i) => replacePrefix(l, [bulletRe, numberedRe], all ? '' : `${i + 1}. `)),
  )
}

export const toggleQuote: FormatCommand = (view) => {
  const lines = targets(selectedLines(view.state))
  const all = lines.every((l) => quoteRe.test(l.text))
  return dispatchChanges(
    view,
    lines.map((l) => replacePrefix(l, [quoteRe], all ? '' : '> ')),
  )
}

/** Swaps the first matching prefix after the indent for `insert`, keeping the indent. */
function replacePrefix(l: LineInfo, res: RegExp[], insert: string) {
  for (const re of res) {
    const m = re.exec(l.text)
    if (m) {
      const indent = m[1]?.length ?? 0
      return { from: l.from + indent, to: l.from + m[0].length, insert }
    }
  }
  const indent = /^\s*/.exec(l.text)?.[0].length ?? 0
  return { from: l.from + indent, insert }
}

export const formatCommands: Record<FormatId, FormatCommand> = {
  h1: toggleHeading1,
  h2: toggleHeading2,
  h3: toggleHeading3,
  bold: toggleBold,
  italic: toggleItalic,
  strike: toggleStrikethrough,
  code: toggleInlineCode,
  link: insertLink,
  bullet: toggleBulletList,
  numbered: toggleNumberedList,
  quote: toggleQuote,
}

export const formatTitles: Record<FormatId, string> = {
  h1: 'Heading 1',
  h2: 'Heading 2',
  h3: 'Heading 3',
  bold: 'Bold',
  italic: 'Italic',
  strike: 'Strikethrough',
  code: 'Inline code',
  link: 'Link',
  bullet: 'Bullet list',
  numbered: 'Numbered list',
  quote: 'Quote',
}

/** Editor-scoped shortcuts for the formats that have one. */
export const formatShortcuts: Partial<Record<FormatId, string>> = {
  bold: 'Mod+B',
  italic: 'Mod+I',
  link: 'Mod+K',
}
