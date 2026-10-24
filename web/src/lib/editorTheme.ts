import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import type { Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { tags as t } from '@lezer/highlight'

/**
 * CodeMirror theme built on the app's CSS variables, so colors follow the
 * light and dark theme. The dark flag only tells CodeMirror which base
 * styles (selection layer, search panel) to use.
 */
export function editorTheme(dark: boolean): Extension {
  return [
    EditorView.theme(
      {
        '&': {
          height: '100%',
          color: 'var(--foreground)',
          backgroundColor: 'var(--background)',
          fontSize: '0.8125rem',
        },
        '&.cm-focused': { outline: 'none' },
        '.cm-scroller': {
          overflow: 'auto',
          fontFamily: 'var(--font-mono)',
          lineHeight: '1.6',
        },
        '.cm-content': { caretColor: 'var(--foreground)', padding: '0.75rem 0' },
        '.cm-line': { padding: '0 1rem' },
        '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--foreground)' },
        '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection':
          { backgroundColor: 'color-mix(in oklab, var(--ring) 30%, transparent)' },
        '.cm-activeLine': { backgroundColor: 'color-mix(in oklab, var(--accent) 50%, transparent)' },
        '.cm-gutters': {
          backgroundColor: 'var(--background)',
          color: 'var(--muted-foreground)',
          border: 'none',
        },
        '.cm-activeLineGutter': {
          backgroundColor: 'transparent',
          color: 'var(--foreground)',
        },
      },
      { dark },
    ),
    syntaxHighlighting(
      HighlightStyle.define([
        { tag: [t.heading, t.heading1, t.heading2, t.heading3], fontWeight: '600', color: 'var(--foreground)' },
        { tag: t.strong, fontWeight: '700' },
        { tag: t.emphasis, fontStyle: 'italic' },
        { tag: t.strikethrough, textDecoration: 'line-through' },
        { tag: [t.link, t.url], color: dark ? 'oklch(0.75 0.12 250)' : 'oklch(0.5 0.15 255)' },
        { tag: t.monospace, color: dark ? 'oklch(0.78 0.1 160)' : 'oklch(0.48 0.1 160)' },
        { tag: t.quote, color: 'var(--muted-foreground)', fontStyle: 'italic' },
        { tag: [t.processingInstruction, t.meta, t.contentSeparator], color: 'var(--muted-foreground)' },
        { tag: t.comment, color: 'var(--muted-foreground)' },
      ]),
    ),
  ]
}
