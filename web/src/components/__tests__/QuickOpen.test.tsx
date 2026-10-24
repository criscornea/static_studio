import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useProjectStore } from '@/stores/project'
import { usePageStore } from '@/stores/page'
import { useUiStore } from '@/stores/ui'
import { QuickOpen } from '../QuickOpen'

const load = vi.fn<(path: string) => Promise<void>>()

beforeEach(() => {
  load.mockReset()
  usePageStore.setState({ load } as never)
  useProjectStore.setState({
    tree: {
      name: 'content',
      path: 'content',
      isDir: true,
      children: [
        {
          name: 'a.md',
          path: 'content/a.md',
          isDir: false,
          fields: { title: 'Alpha', draft: false },
        },
      ],
    },
  })
  useUiStore.setState({ quickOpenOpen: false, paletteOpen: false })
  vi.stubGlobal('fetch', () =>
    Promise.resolve(
      new Response(
        JSON.stringify({
          hits: [{ path: 'content/b.md', title: 'Beta', draft: false }],
          truncated: false,
        }),
        { status: 200 },
      ),
    ),
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('QuickOpen', () => {
  it('lists pages from the tree and opens the chosen one', async () => {
    const user = userEvent.setup()
    render(<QuickOpen />)
    act(() => useUiStore.getState().setQuickOpenOpen(true))

    await user.click(await screen.findByText('Alpha'))

    expect(load).toHaveBeenCalledWith('content/a.md')
    expect(useUiStore.getState().quickOpenOpen).toBe(false)
  })

  it('searches by what is typed', async () => {
    const user = userEvent.setup()
    render(<QuickOpen />)
    act(() => useUiStore.getState().setQuickOpenOpen(true))

    await user.type(await screen.findByPlaceholderText('Open page...'), 'bet')
    await user.click(await screen.findByText('Beta'))

    expect(load).toHaveBeenCalledWith('content/b.md')
  })
})
