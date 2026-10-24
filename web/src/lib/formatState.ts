import { ensureSyntaxTree } from '@codemirror/language'
import type { EditorState } from '@codemirror/state'
import type { SyntaxNode } from '@lezer/common'

export type FormatId =
  | 'h1'
  | 'h2'
  | 'h3'
  | 'bold'
  | 'italic'
  | 'strike'
  | 'code'
  | 'link'
  | 'bullet'
  | 'numbered'
  | 'quote'

/** Syntax tree node names that stand for each inline or block format. */
const nodeNames: Record<FormatId, string[]> = {
  h1: ['ATXHeading1', 'SetextHeading1'],
  h2: ['ATXHeading2', 'SetextHeading2'],
  h3: ['ATXHeading3'],
  bold: ['StrongEmphasis'],
  italic: ['Emphasis'],
  strike: ['Strikethrough'],
  code: ['InlineCode'],
  link: ['Link'],
  bullet: ['BulletList'],
  numbered: ['OrderedList'],
  quote: ['Blockquote'],
}

function tree(state: EditorState) {
  return ensureSyntaxTree(state, state.doc.length, 50)
}

/**
 * Innermost ancestor named one of `names` that spans [from, to]. For lists the
 * innermost list of either kind decides, so a numbered list inside a bullet
 * list counts as numbered only.
 */
export function enclosing(
  state: EditorState,
  from: number,
  to: number,
  names: string[],
): SyntaxNode | null {
  const t = tree(state)
  if (!t) return null
  for (const side of [1, -1] as const) {
    for (let n: SyntaxNode | null = t.resolveInner(from, side); n; n = n.parent) {
      if (names.includes(n.name) && n.from <= from && n.to >= to) return n
    }
  }
  return null
}

function innermostList(state: EditorState, from: number, to: number): 'bullet' | 'numbered' | null {
  const t = tree(state)
  if (!t) return null
  for (const side of [1, -1] as const) {
    for (let n: SyntaxNode | null = t.resolveInner(from, side); n; n = n.parent) {
      if (n.from <= from && n.to >= to) {
        if (n.name === 'BulletList') return 'bullet'
        if (n.name === 'OrderedList') return 'numbered'
      }
    }
  }
  return null
}

/** Formats active for the main selection, read from the markdown syntax tree. */
export function activeFormats(state: EditorState): Set<FormatId> {
  const { from, to } = state.selection.main
  const active = new Set<FormatId>()
  for (const id of Object.keys(nodeNames) as FormatId[]) {
    if (id === 'bullet' || id === 'numbered') continue
    if (enclosing(state, from, to, nodeNames[id])) active.add(id)
  }
  const list = innermostList(state, from, to)
  if (list) active.add(list)
  return active
}
