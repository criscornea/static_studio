import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it } from 'vitest'

import { useLayoutStore } from '@/stores/layout'
import { useUiStore } from '@/stores/ui'
import { CommandPalette } from '../CommandPalette'

beforeEach(() => {
  useLayoutStore.setState({ sidebarCollapsed: false })
  useUiStore.setState({ paletteOpen: false, quickOpenOpen: false })
})

describe('CommandPalette', () => {
  it('stays closed until opened', () => {
    render(<CommandPalette />)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('filters commands and runs the chosen one', async () => {
    const user = userEvent.setup()
    render(<CommandPalette />)
    act(() => useUiStore.getState().setPaletteOpen(true))

    await user.type(await screen.findByPlaceholderText('Type a command...'), 'sidebar')
    expect(screen.queryByText('Build site')).toBeNull()

    await user.click(screen.getByText('Toggle sidebar'))

    expect(useLayoutStore.getState().sidebarCollapsed).toBe(true)
    expect(useUiStore.getState().paletteOpen).toBe(false)
  })

  it('says so when nothing matches', async () => {
    const user = userEvent.setup()
    render(<CommandPalette />)
    act(() => useUiStore.getState().setPaletteOpen(true))

    await user.type(await screen.findByPlaceholderText('Type a command...'), 'zzzz')

    expect(screen.getByText('No command matches.')).toBeInTheDocument()
  })
})
