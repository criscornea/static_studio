import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ContentNode, SearchResults } from '@/api/types'
import { defaultView } from '@/lib/tree'
import { useProjectStore } from '@/stores/project'
import { useSearchStore } from '@/stores/search'
import { useViewStore } from '@/stores/view'
import { useUiStore } from '@/stores/ui'
import { AppDialogs } from '../AppDialogs'
import { Sidebar } from '../Sidebar'

const tree: ContentNode = {
  name: 'content',
  path: 'content',
  isDir: true,
  children: [
    {
      name: 'live.md',
      path: 'content/live.md',
      isDir: false,
      fields: { title: 'Live Post', draft: false },
    },
    {
      name: 'wip.md',
      path: 'content/wip.md',
      isDir: false,
      fields: { title: 'Work In Progress', draft: true },
    },
  ],
}

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
  useProjectStore.setState({ tree })
  useViewStore.setState(defaultView)
  useSearchStore.getState().clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Sidebar', () => {
  it('lists every page by default', () => {
    render(<Sidebar />)

    expect(screen.getByText('Live Post')).toBeInTheDocument()
    expect(screen.getByText('Work In Progress')).toBeInTheDocument()
  })

  it('shows only drafts when filtered, and offers a way back when nothing matches', async () => {
    render(<Sidebar />)

    useViewStore.getState().setStatus('draft')
    expect(await screen.findByText('Work In Progress')).toBeInTheDocument()
    expect(screen.queryByText('Live Post')).not.toBeInTheDocument()

    useViewStore.getState().setTag('missing')
    expect(await screen.findByText('No pages match this filter.')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Show all pages' }))
    expect(await screen.findByText('Live Post')).toBeInTheDocument()
  })

  it('replaces the tree with search results while a query is typed, and brings it back', async () => {
    const fetch = stubSearch(found)
    render(<Sidebar />)

    await userEvent.type(screen.getByRole('searchbox', { name: 'Search pages' }), 'golang')

    expect(await screen.findByText('content/live.md')).toBeInTheDocument()
    expect(screen.queryByText('Work In Progress')).not.toBeInTheDocument()
    expect(screen.getByText('golang', { selector: 'mark' })).toBeInTheDocument()
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(String(fetch.mock.calls[0]?.[0])).toBe('/api/search?q=golang')

    await userEvent.click(screen.getByRole('button', { name: 'Clear search' }))

    expect(await screen.findByText('Work In Progress')).toBeInTheDocument()
  })

  it('says so when nothing matches', async () => {
    stubSearch({ hits: [], truncated: false })
    render(<Sidebar />)

    await userEvent.type(screen.getByRole('searchbox'), 'zzz')

    expect(await screen.findByText('No pages match “zzz”.')).toBeInTheDocument()
  })

  it('clears the search on Escape', async () => {
    stubSearch(found)
    render(<Sidebar />)
    const box = screen.getByRole('searchbox')

    await userEvent.type(box, 'golang{Escape}')

    await waitFor(() => expect(box).toHaveValue(''))
  })

  it('focuses the search box when the focus command asks', () => {
    render(<Sidebar />)

    act(() => useUiStore.getState().requestSearchFocus())

    expect(screen.getByRole('searchbox')).toHaveFocus()
  })
})

describe('Sidebar new page', () => {
  it('opens the new page dialog from the plus button', async () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(new Response(JSON.stringify({ archetypes: [] }), { status: 200 })),
    )
    render(
      <>
        <Sidebar />
        <AppDialogs />
      </>,
    )

    await userEvent.click(screen.getByRole('button', { name: 'New page' }))

    expect(await screen.findByLabelText('Name')).toBeInTheDocument()
  })
})

describe('Sidebar empty and loading states', () => {
  it('shows skeletons while the tree loads', () => {
    useProjectStore.setState({ tree: null })
    render(<Sidebar />)
    expect(screen.getByRole('status')).toHaveTextContent('Loading pages')
  })

  it('offers a new page when the content dir is empty', async () => {
    useProjectStore.setState({
      tree: { name: 'content', path: 'content', isDir: true, children: [] },
    })
    render(<Sidebar />)
    expect(screen.getByText('No content files yet.')).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole('button', { name: 'New page' }).at(-1)!)
    expect(useUiStore.getState().newPage).not.toBeNull()
  })
})

describe('Sidebar with several content dirs', () => {
  afterEach(() => useProjectStore.setState({ roots: [] }))

  it('renders one labelled tree per content dir', () => {
    const page = (path: string, title: string): ContentNode => ({
      name: path.split('/').pop()!,
      path,
      isDir: false,
      fields: { title, draft: false },
    })
    const dir = (path: string, child: ContentNode): ContentNode => ({
      name: path,
      path,
      isDir: true,
      children: [child],
    })
    const en = dir('content', page('content/a.md', 'English page'))
    const de = dir('inhalt', page('inhalt/b.md', 'German page'))
    useProjectStore.setState({
      tree: en,
      roots: [
        { dir: 'content', root: en },
        { dir: 'inhalt', lang: 'de', root: de },
      ],
    })
    render(<Sidebar />)

    expect(screen.getAllByTestId('content-root')).toHaveLength(2)
    expect(screen.getByRole('tree', { name: 'Pages in inhalt' })).toBeInTheDocument()
    expect(screen.getByText('de')).toBeInTheDocument()
    expect(screen.getByText('English page')).toBeInTheDocument()
    expect(screen.getByText('German page')).toBeInTheDocument()
  })
})
