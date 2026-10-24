import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { history, undo } from '@codemirror/commands'
import { EditorSelection, EditorState, type Transaction, type TransactionSpec } from '@codemirror/state'
import type { EditorView } from '@codemirror/view'
import { describe, expect, it } from 'vitest'

import {
  insertLink,
  toggleBold,
  toggleBulletList,
  toggleHeading1,
  toggleHeading2,
  toggleInlineCode,
  toggleItalic,
  toggleNumberedList,
  toggleQuote,
  toggleStrikethrough,
  type FormatCommand,
} from '../editorFormat'
import { activeFormats } from '../formatState'

/** The commands only need state and dispatch, so skip the DOM (jsdom cannot measure text). */
function make(doc: string, anchor: number, head = anchor) {
  const v = {
    state: EditorState.create({
      doc,
      selection: EditorSelection.single(anchor, head),
      extensions: [markdown({ base: markdownLanguage }), history()],
    }),
    dispatch(...args: (Transaction | TransactionSpec)[]) {
      v.state = (args[0] as Transaction).state ?? v.state.update(...(args as TransactionSpec[])).state
    },
  }
  return v as unknown as EditorView
}

function sel(v: EditorView) {
  const r = v.state.selection.main
  return [r.anchor, r.head]
}

describe.each([
  ['bold', toggleBold, '**'],
  ['italic', toggleItalic, '*'],
  ['strikethrough', toggleStrikethrough, '~~'],
  ['inline code', toggleInlineCode, '`'],
] as [string, FormatCommand, string][])('%s', (_name, cmd, m) => {
  it('wraps the selection and keeps it selected', () => {
    const v = make('a word b', 2, 6)
    cmd(v)
    expect(v.state.doc.toString()).toBe(`a ${m}word${m} b`)
    expect(sel(v)).toEqual([2 + m.length, 6 + m.length])
  })

  it('unwraps when already wrapped', () => {
    const v = make(`a ${m}word${m} b`, 2 + m.length, 6 + m.length)
    cmd(v)
    expect(v.state.doc.toString()).toBe('a word b')
    expect(sel(v)).toEqual([2, 6])
  })

  it('puts the cursor between markers for an empty selection', () => {
    const v = make('ab', 1)
    cmd(v)
    expect(v.state.doc.toString()).toBe(`a${m}${m}b`)
    expect(sel(v)).toEqual([1 + m.length, 1 + m.length])
  })

  it('unwraps from a bare cursor inside the text', () => {
    const v = make(`${m}word${m}`, m.length + 2)
    cmd(v)
    expect(v.state.doc.toString()).toBe('word')
    expect(sel(v)).toEqual([2, 2])
  })

  it('is one undoable transaction', () => {
    const v = make('a word b', 2, 6)
    cmd(v)
    expect(undo(v)).toBe(true)
    expect(v.state.doc.toString()).toBe('a word b')
  })
})

describe('bold inside italic', () => {
  it('only removes the matching markers', () => {
    const v = make('***x***', 3)
    toggleBold(v)
    expect(v.state.doc.toString()).toBe('*x*')
  })
})

describe('headings', () => {
  it('sets, changes and removes the level', () => {
    const v = make('title', 2)
    toggleHeading1(v)
    expect(v.state.doc.toString()).toBe('# title')
    toggleHeading2(v)
    expect(v.state.doc.toString()).toBe('## title')
    toggleHeading2(v)
    expect(v.state.doc.toString()).toBe('title')
  })

  it('applies to every selected line but skips blank ones', () => {
    const v = make('a\n\nb', 0, 4)
    toggleHeading1(v)
    expect(v.state.doc.toString()).toBe('# a\n\n# b')
    toggleHeading1(v)
    expect(v.state.doc.toString()).toBe('a\n\nb')
  })

  it('works on an empty line', () => {
    const v = make('', 0)
    toggleHeading1(v)
    expect(v.state.doc.toString()).toBe('# ')
  })
})

describe('lists and quotes', () => {
  it('toggles bullets on every selected line', () => {
    const v = make('a\nb\nc', 0, 5)
    toggleBulletList(v)
    expect(v.state.doc.toString()).toBe('- a\n- b\n- c')
    toggleBulletList(v)
    expect(v.state.doc.toString()).toBe('a\nb\nc')
  })

  it('numbers lines and converts from bullets', () => {
    const v = make('- a\n- b', 0, 7)
    toggleNumberedList(v)
    expect(v.state.doc.toString()).toBe('1. a\n2. b')
    toggleNumberedList(v)
    expect(v.state.doc.toString()).toBe('a\nb')
  })

  it('keeps indentation', () => {
    const v = make('  a', 3)
    toggleBulletList(v)
    expect(v.state.doc.toString()).toBe('  - a')
    toggleBulletList(v)
    expect(v.state.doc.toString()).toBe('  a')
  })

  it('toggles quotes and ignores the line a selection only touches at its start', () => {
    const v = make('a\nb', 0, 2)
    toggleQuote(v)
    expect(v.state.doc.toString()).toBe('> a\nb')
    toggleQuote(v)
    expect(v.state.doc.toString()).toBe('a\nb')
  })

  it('is one undoable transaction for many lines', () => {
    const v = make('a\nb\nc', 0, 5)
    toggleQuote(v)
    undo(v)
    expect(v.state.doc.toString()).toBe('a\nb\nc')
  })
})

describe('link', () => {
  it('wraps the selection and selects the url', () => {
    const v = make('see docs now', 4, 8)
    insertLink(v)
    expect(v.state.doc.toString()).toBe('see [docs](url) now')
    expect(v.state.sliceDoc(...(sel(v).sort((a, b) => a - b) as [number, number]))).toBe('url')
  })

  it('inserts a placeholder link for an empty selection', () => {
    const v = make('', 0)
    insertLink(v)
    expect(v.state.doc.toString()).toBe('[link](url)')
    expect(sel(v)).toEqual([7, 10])
  })

  it('selects the url of an existing link', () => {
    const v = make('[a](http://x.y)', 1)
    insertLink(v)
    expect(v.state.doc.toString()).toBe('[a](http://x.y)')
    expect(v.state.sliceDoc(sel(v)[0], sel(v)[1])).toBe('http://x.y')
  })

  it('uses a selected address as the target', () => {
    const v = make('http://x.y', 0, 10)
    insertLink(v)
    expect(v.state.doc.toString()).toBe('[](http://x.y)')
    expect(sel(v)).toEqual([1, 1])
  })
})

describe('activeFormats', () => {
  const at = (doc: string, a: number, b = a) => [...activeFormats(make(doc, a, b).state)].sort()

  it('detects inline formats', () => {
    expect(at('**bo**', 3)).toEqual(['bold'])
    expect(at('*it*', 2)).toEqual(['italic'])
    expect(at('~~s~~', 3)).toEqual(['strike'])
    expect(at('`c`', 2)).toEqual(['code'])
    expect(at('[a](u)', 1)).toEqual(['link'])
    expect(at('***x***', 4)).toEqual(['bold', 'italic'])
  })

  it('detects headings, lists and quotes', () => {
    expect(at('## t', 4)).toEqual(['h2'])
    expect(at('- a', 3)).toEqual(['bullet'])
    expect(at('1. a', 4)).toEqual(['numbered'])
    expect(at('> q', 3)).toEqual(['quote'])
    expect(at('- a\n  1. b', 10)).toEqual(['numbered'])
  })

  it('reports nothing for plain text and for a selection leaving the format', () => {
    expect(at('plain', 2)).toEqual([])
    expect(at('**bo** x', 3, 8)).toEqual([])
  })
})
