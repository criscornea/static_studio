import { act, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { buildCommands } from '@/lib/commands'
import { formatShortcut } from '@/lib/shortcuts'
import { useUiStore } from '@/stores/ui'
import { ShortcutsDialog } from '../ShortcutsDialog'

describe('ShortcutsDialog', () => {
  it('lists every bound command with its shortcut', () => {
    render(<ShortcutsDialog />)
    expect(screen.queryByRole('dialog')).toBeNull()

    act(() => useUiStore.getState().setShortcutsOpen(true))

    const bound = buildCommands().filter((c) => c.shortcut)
    expect(bound.length).toBeGreaterThan(5)
    for (const c of bound) {
      expect(screen.getAllByText(c.title).length).toBeGreaterThan(0)
      expect(screen.getAllByText(formatShortcut(c.shortcut ?? '')).length).toBeGreaterThan(0)
    }
  })
})
