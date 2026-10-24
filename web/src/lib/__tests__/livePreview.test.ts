import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { EditorSelection, EditorState } from '@codemirror/state'
import { describe, expect, it } from 'vitest'

import { livePreviewDecorations } from '../livePreview'

interface Found {
  from: number
  to: number
  kind: string
  cls?: string
  widget?: string
}

/** All decorations for doc with the cursor at `at` (end of doc by default). */
export function decorate(doc: string, at?: number, head = at, visible?: { from: number; to: number }[]) {
  const state = EditorState.create({
    doc,
    selection: EditorSelection.single(at ?? doc.length, head ?? at ?? doc.length),
    extensions: [markdown({ base: markdownLanguage })],
  })
  const out: Found[] = []
  livePreviewDecorations(state, visible).between(0, doc.length, (from, to, d) => {
    const spec = d.spec as { kind: string; class?: string; widget?: { label: string } }
    out.push({ from, to, kind: spec.kind, cls: spec.class, widget: spec.widget?.label })
  })
  return { state, found: out }
}

const hidden = (f: Found[]) => f.filter((d) => d.kind === 'hide').map((d): [number, number] => [d.from, d.to])

describe('headings, emphasis and code', () => {
  const doc = 'text\n\n# Title\n\n**b** _i_ ~~s~~ `c`'

  it('hides markup away from the cursor and styles the content', () => {
    const { found } = decorate(doc, 0)
    expect(hidden(found)).toEqual([
      [6, 8], // "# "
      [15, 17], // **
      [18, 20],
      [21, 22], // _
      [23, 24],
      [25, 27], // ~~
      [28, 30],
      [31, 32], // `
      [33, 34],
    ])
    expect(found.filter((d) => d.cls === 'cm-lp-h1')).toHaveLength(1)
    for (const c of ['cm-lp-strong', 'cm-lp-em', 'cm-lp-strike', 'cm-lp-code']) {
      expect(found.filter((d) => d.cls === c)).toHaveLength(1)
    }
  })

  it('reveals markup on the cursor line only', () => {
    const { found } = decorate(doc, 8) // inside the heading
    expect(hidden(found).some(([f]) => f === 6)).toBe(false)
    expect(hidden(found).some(([f]) => f === 15)).toBe(true)
  })

  it('reveals every line a selection touches', () => {
    const { found } = decorate(doc, 0, doc.length)
    expect(hidden(found)).toEqual([])
  })

  it('assigns heading levels', () => {
    const { found } = decorate('## Two\n\n### Three\n\nx', 19)
    expect(found.filter((d) => d.kind === 'line').map((d) => d.cls)).toEqual(['cm-lp-h2', 'cm-lp-h3'])
  })

  it('leaves code blocks and fence marks as source', () => {
    const { found } = decorate('```js\n**x** # y\n```\n\nend', 21)
    expect(found).toEqual([])
  })

  it('walks only visible ranges', () => {
    const d = '# A\n\n# B\n\n# C\n\nend'
    const { found } = decorate(d, d.length, d.length, [{ from: 5, to: 8 }])
    expect(found.filter((f) => f.kind === 'line')).toHaveLength(1)
  })

  it('never changes the document', () => {
    const a = decorate(doc, 0)
    const b = decorate(doc, 8)
    expect(a.state.doc.toString()).toBe(doc)
    expect(b.state.doc.toString()).toBe(doc)
  })
})

describe('links and quotes', () => {
  const doc = 'end\n\n[site](https://x.io) and ![img](a.png)\n\n> quoted\n> more\n\n[ref][1]'

  it('shows only the link text off the cursor line', () => {
    const { found } = decorate(doc, 0)
    // "[" at 5, "](https://x.io)" at 10..25
    expect(hidden(found)).toContainEqual([5, 6])
    expect(hidden(found)).toContainEqual([10, 25])
    expect(found.filter((d) => d.cls === 'cm-lp-link')).toEqual([
      { from: 6, to: 10, kind: 'mark', cls: 'cm-lp-link', widget: undefined },
    ])
  })

  it('reveals the link source on its line', () => {
    const { found } = decorate(doc, 8)
    expect(hidden(found).some(([f]) => f === 5 || f === 10)).toBe(false)
  })

  it('leaves images and reference links as source', () => {
    const { found } = decorate(doc, 0)
    expect(hidden(found).filter(([f]) => f > 26 && f < 45)).toEqual([])
    expect(hidden(found).some(([f]) => f >= doc.indexOf('[ref]'))).toBe(false)
  })

  it('hides quote marks and marks every quote line', () => {
    const { found } = decorate(doc, 0)
    const q = doc.indexOf('> quoted')
    expect(hidden(found)).toContainEqual([q, q + 2])
    expect(found.filter((d) => d.cls === 'cm-lp-quote')).toHaveLength(2)
  })

  it('shows the quote mark on the cursor line', () => {
    const q = doc.indexOf('> quoted')
    const { found } = decorate(doc, q + 3)
    expect(hidden(found).some(([f]) => f === q)).toBe(false)
    expect(hidden(found).some(([f]) => f === q + 9)).toBe(true)
  })
})

describe('list markers and rules', () => {
  const doc = 'end\n\n- one\n- two\n\n1. first\n2. second\n\n---\n\n***\n\nx'
  const widgets = (f: Found[]) => f.filter((d) => d.kind === 'widget').map((d) => [d.from, d.to, d.widget])

  it('replaces markers with bullet and number widgets', () => {
    const { found } = decorate(doc, 0)
    const w = widgets(found)
    expect(w).toContainEqual([doc.indexOf('- one'), doc.indexOf('- one') + 1, '\u2022'])
    expect(w).toContainEqual([doc.indexOf('- two'), doc.indexOf('- two') + 1, '\u2022'])
    expect(w).toContainEqual([doc.indexOf('1.'), doc.indexOf('1.') + 2, '1.'])
    expect(w).toContainEqual([doc.indexOf('2.'), doc.indexOf('2.') + 2, '2.'])
  })

  it('shows the raw marker on the cursor line', () => {
    const { found } = decorate(doc, doc.indexOf('- one') + 3)
    expect(widgets(found).some(([f]) => f === doc.indexOf('- one'))).toBe(false)
    expect(widgets(found).some(([f]) => f === doc.indexOf('- two'))).toBe(true)
  })

  it('renders both rule styles as a line', () => {
    const { found } = decorate(doc, 0)
    const rules = found.filter((d) => d.kind === 'widget' && d.widget === '---')
    expect(rules.map((r) => [r.from, r.to])).toEqual([
      [doc.indexOf('---'), doc.indexOf('---') + 3],
      [doc.indexOf('***'), doc.indexOf('***') + 3],
    ])
  })

  it('reveals a rule under the cursor', () => {
    const { found } = decorate(doc, doc.indexOf('---') + 1)
    expect(found.filter((d) => d.widget === '---')).toHaveLength(1)
  })

  it('does not change the document', () => {
    expect(decorate(doc, 0).state.doc.toString()).toBe(doc)
  })
})
