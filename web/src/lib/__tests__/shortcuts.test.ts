import { describe, expect, it } from 'vitest'

import { formatShortcut, matchesShortcut } from '../shortcuts'

function key(init: KeyboardEventInit): KeyboardEvent {
  return new KeyboardEvent('keydown', init)
}

describe('matchesShortcut', () => {
  it('accepts Ctrl or Cmd for Mod', () => {
    expect(matchesShortcut(key({ key: 'b', ctrlKey: true }), 'Mod+B')).toBe(true)
    expect(matchesShortcut(key({ key: 'b', metaKey: true }), 'Mod+B')).toBe(true)
  })

  it('requires the exact modifier set', () => {
    expect(matchesShortcut(key({ key: 'b' }), 'Mod+B')).toBe(false)
    expect(matchesShortcut(key({ key: 'p', ctrlKey: true, shiftKey: true }), 'Mod+P')).toBe(false)
    expect(matchesShortcut(key({ key: 'P', ctrlKey: true, shiftKey: true }), 'Mod+Shift+P')).toBe(
      true,
    )
  })

  it('matches by physical key when Alt changes the character', () => {
    const e = key({ key: '∫', code: 'KeyB', metaKey: true, altKey: true })
    expect(matchesShortcut(e, 'Mod+Alt+B')).toBe(true)
  })

  it('matches bare keys', () => {
    expect(matchesShortcut(key({ key: 'F2' }), 'F2')).toBe(true)
  })
})

describe('formatShortcut', () => {
  it('uses words on other platforms and symbols on macOS', () => {
    expect(formatShortcut('Mod+Shift+P', false)).toBe('Ctrl+Shift+P')
    expect(formatShortcut('Mod+Shift+P', true)).toBe('⌘⇧P')
    expect(formatShortcut('F2', false)).toBe('F2')
  })
})
