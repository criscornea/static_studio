import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { SearchResults } from '@/api/types'
import { useSearchStore } from '@/stores/search'
import { useUiStore } from '@/stores/ui'
import { SearchPanel } from '../SearchPanel'

const found: SearchResults = {
  hits: [
    {
      path: 'content/live.md',
      title: 'Live Post',
      draft: false,
      snippet: 'all about golang here',
    },
  ],
  truncated: false,
}

function stubSearch(results: SearchResults) {
  const fetch = vi.fn<(input: RequestInfo) => Promise<Response>>(() =>
    Promise.resolve(new Response(JSON.stringify(results), { status: 200 })),
  )
  vi.stubGlobal('fetch', fetch)
  return fetch
}

beforeEach(() => {
  useSearchStore.getState().clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('SearchPanel', () => {
  it('replaces the tree with search results while a query is typed, and brings it back', async () => {
    const fetch = stubSearch(found)
    render(<SearchPanel />)

    await userEvent.type(screen.getByRole('searchbox', { name: 'Search pages' }), 'golang')

    expect(await screen.findByText('content/live.md')).toBeInTheDocument()
    expect(screen.getByText('golang', { selector: 'mark' })).toBeInTheDocument()
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(String(fetch.mock.calls[0]?.[0])).toBe('/api/search?q=golang')

    await userEvent.click(screen.getByRole('button', { name: 'Clear search' }))

    expect(screen.queryByText('content/live.md')).not.toBeInTheDocument()
  })

  it('says so when nothing matches', async () => {
    stubSearch({ hits: [], truncated: false })
    render(<SearchPanel />)

    await userEvent.type(screen.getByRole('searchbox'), 'zzz')

    expect(await screen.findByText('No pages match “zzz”.')).toBeInTheDocument()
  })

  it('clears the search on Escape', async () => {
    stubSearch(found)
    render(<SearchPanel />)
    const box = screen.getByRole('searchbox')

    await userEvent.type(box, 'golang{Escape}')

    await waitFor(() => expect(box).toHaveValue(''))
  })

  it('focuses the search box when the focus command asks', () => {
    render(<SearchPanel />)

    act(() => useUiStore.getState().requestSearchFocus())

    expect(screen.getByRole('searchbox')).toHaveFocus()
  })

  it('keeps the query and results when remounted', async () => {
    stubSearch(found)
    const first = render(<SearchPanel />)
    await userEvent.type(screen.getByRole('searchbox'), 'golang')
    expect(await screen.findByText('content/live.md')).toBeInTheDocument()
    first.unmount()

    render(<SearchPanel />)

    expect(screen.getByRole('searchbox')).toHaveValue('golang')
    expect(screen.getByText('content/live.md')).toBeInTheDocument()
  })

  it('restores the saved scroll position', () => {
    useSearchStore.setState({ scrollTop: 40 })
    const { unmount } = render(<SearchPanel />)
    expect(screen.getByTestId('search-scroll').scrollTop).toBe(40)
    unmount()
  })
})
